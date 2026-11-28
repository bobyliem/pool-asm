package whatsapp

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"gorm.io/gorm"
)

// ==================== DATA STRUCTURES ====================

// SolarUnit represents a row from dt_solar_unit used for matching
type SolarUnit struct {
	ID       int     `gorm:"column:id"`
	Unit     string  `gorm:"column:unit"`
	Alias    string  `gorm:"column:alias"`
	Konsumsi float64 `gorm:"column:konsumsi"`
	Tangki   string  `gorm:"column:tangki"`
}

// ParsedSolarReport represents a single parsed solar report from WA text
type ParsedSolarReport struct {
	JenisDt   string    // "trigger" or "isi"
	Tanggal   time.Time // parsed date
	Unit      string    // matched unit name from dt_solar_unit
	RawUnit   string    // original unit text from WA
	Hm        float64   // hour meter
	IAwal     string    // raw input awal (e.g. "33 cm")
	IIsi      string    // raw input isi (e.g. "10 jerigen")
	IAkhir    string    // raw input akhir (e.g. "59 cm")
	Awal      float64   // converted to liter
	Isi       float64   // converted to liter
	Akhir     float64   // converted to liter
	Konsumsi  float64   // liter per cm from dt_solar_unit
	InWa      *time.Time // WA message timestamp (nullable)
	ParseNote string    // any warnings during parse
}

// ParseResult holds the overall result of parsing WA text
type ParseResult struct {
	Reports []ParsedSolarReport
	Errors  []string
}

// ==================== MAIN PARSER ====================

// ParseSolarWA is the main entry point: takes raw WA text and returns parsed reports
func ParseSolarWA(db *gorm.DB, rawText string, waTimestamp *time.Time) ParseResult {
	result := ParseResult{}

	// 1. Load unit master from DB
	units := loadSolarUnits(db)
	if len(units) == 0 {
		result.Errors = append(result.Errors, "Gagal memuat data dt_solar_unit")
		return result
	}

	// 2. Split text into individual report blocks
	blocks := splitReportBlocks(rawText)
	if len(blocks) == 0 {
		result.Errors = append(result.Errors, "Tidak ditemukan blok laporan solar dalam teks")
		return result
	}

	// 3. Parse each block
	for i, block := range blocks {
		report, err := parseOneBlock(block, units)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Blok #%d: %v", i+1, err))
			continue
		}
		report.InWa = waTimestamp
		result.Reports = append(result.Reports, report)
	}

	return result
}

// ==================== UNIT LOADING ====================

func loadSolarUnits(db *gorm.DB) []SolarUnit {
	var units []SolarUnit
	db.Table("dt_solar_unit").
		Select("id, unit, COALESCE(alias, '') as alias, konsumsi, tangki").
		Where("tangki = 'No'").
		Find(&units)
	return units
}

// ==================== BLOCK SPLITTING ====================

// splitReportBlocks splits a large WA text into individual report blocks
func splitReportBlocks(text string) []string {
	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")

	var blocks []string
	var currentBlock []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip separator lines
		if trimmed == "" || regexp.MustCompile(`^-{3,}$`).MatchString(trimmed) {
			continue
		}

		// Check if this line starts a new block
		if isBlockStart(trimmed) {
			if len(currentBlock) > 0 {
				blocks = append(blocks, strings.Join(currentBlock, "\n"))
			}
			currentBlock = []string{trimmed}
		} else {
			currentBlock = append(currentBlock, trimmed)
		}
	}

	// Don't forget the last block
	if len(currentBlock) > 0 {
		blocks = append(blocks, strings.Join(currentBlock, "\n"))
	}

	return blocks
}

// isBlockStart detects whether a line starts a new solar report block
func isBlockStart(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "laporan solar")
}

// ==================== SINGLE BLOCK PARSING ====================

func parseOneBlock(block string, units []SolarUnit) (ParsedSolarReport, error) {
	report := ParsedSolarReport{}

	// Try compact format first (e.g. "01/330/43/cm\nHM:14942, 0")
	if compactReport, ok := parseCompactBlock(block, units); ok {
		// For compact blocks without an explicit date, use today's date
		if compactReport.Tanggal.IsZero() {
			compactReport.Tanggal = time.Now()
		}
		return compactReport, nil
	}

	lines := strings.Split(block, "\n")

	// Extract all fields
	rawUnit := extractUnit(lines)
	rawDate := extractDate(lines)
	rawHm := extractHM(lines)
	rawColokan, colokanContext := extractColokan(lines)
	rawJerigen := extractJerigen(lines)
	rawSesudah := extractSesudah(lines)

	// --- Parse Date ---
	tgl, err := parseDate(rawDate)
	if err != nil {
		return report, fmt.Errorf("gagal parse tanggal '%s': %v", rawDate, err)
	}
	report.Tanggal = tgl

	// --- Match Unit ---
	if rawUnit == "" {
		return report, fmt.Errorf("unit tidak ditemukan dalam blok")
	}
	matched, matchedUnit := matchUnit(rawUnit, units)
	if !matched {
		return report, fmt.Errorf("unit '%s' tidak cocok dengan master data", rawUnit)
	}
	report.Unit = matchedUnit.Unit
	report.RawUnit = rawUnit
	report.Konsumsi = matchedUnit.Konsumsi

	// --- Parse HM ---
	report.Hm = parseNumber(rawHm)

	// --- Determine type (trigger vs isi) and parse values ---
	hasIsi := rawJerigen > 0
	hasSesudah := rawSesudah > 0
	gaIsi := containsGaIsi(block)

	if hasIsi && hasSesudah {
		// JENIS: ISI SOLAR
		report.JenisDt = "isi"

		// i_awal: colokan sebelum isi (CM)
		report.IAwal = fmt.Sprintf("%.0f cm", rawColokan)
		report.Awal = rawColokan * matchedUnit.Konsumsi

		// i_isi: jumlah jerigen
		report.IIsi = fmt.Sprintf("%.0f jerigen", rawJerigen)
		report.Isi = rawJerigen * 20 // 1 jerigen = 20 liter (static)

		// i_akhir: colokan sesudah isi (CM)
		report.IAkhir = fmt.Sprintf("%.0f cm", rawSesudah)
		report.Akhir = rawSesudah * matchedUnit.Konsumsi

	} else if hasIsi && !hasSesudah {
		// ISI tapi sesudah belum diisi — still categorize as isi
		report.JenisDt = "isi"
		report.IAwal = fmt.Sprintf("%.0f cm", rawColokan)
		report.Awal = rawColokan * matchedUnit.Konsumsi
		report.IIsi = fmt.Sprintf("%.0f jerigen", rawJerigen)
		report.Isi = rawJerigen * 20

	} else if gaIsi || (!hasIsi && rawColokan > 0) {
		// JENIS: TRIGGER (cek colokan saja, tidak isi solar)
		report.JenisDt = "trigger"

		report.IAwal = fmt.Sprintf("%.0f cm", rawColokan)
		report.Awal = rawColokan * matchedUnit.Konsumsi

		_ = colokanContext // extra info like "pagi", "malam", etc
	} else {
		// Template kosong: colokan ada tapi jerigen/sesudah template placeholder
		// Check if colokan has a value
		if rawColokan > 0 {
			report.JenisDt = "trigger"
			report.IAwal = fmt.Sprintf("%.0f cm", rawColokan)
			report.Awal = rawColokan * matchedUnit.Konsumsi
		} else {
			return report, fmt.Errorf("data tidak lengkap: tidak ada nilai colokan atau pengisian")
		}
	}

	return report, nil
}

// ==================== FIELD EXTRACTORS ====================

// extractUnit finds the unit name from the block lines
func extractUnit(lines []string) string {
	for _, line := range lines {
		lower := strings.ToLower(line)

		// Pattern: "Unit: xxx" or "Unit:xxx"
		if strings.HasPrefix(lower, "unit") {
			val := line
			if idx := strings.Index(lower, ":"); idx >= 0 {
				val = line[idx+1:]
			} else if idx := strings.Index(lower, " "); idx >= 0 {
				val = line[idx+1:]
			}
			val = strings.TrimSpace(val)
			if val != "" {
				return val
			}
		}

		// Pattern: "SK 200 (04" or "SK200-10" at the start of a line (without label)
		if matchesFreeformUnit(line) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// matchesFreeformUnit detects lines like "SK 200 (04 bleker" that are unit names without a label
func matchesFreeformUnit(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))

	// Skip lines that are clearly other fields
	if strings.HasPrefix(lower, "laporan") || strings.HasPrefix(lower, "tgl") ||
		strings.HasPrefix(lower, "tanggal") || strings.HasPrefix(lower, "hm") ||
		strings.HasPrefix(lower, "colokan") || strings.HasPrefix(lower, "clkn") ||
		strings.HasPrefix(lower, "isi") || strings.HasPrefix(lower, "sesudah") ||
		strings.HasPrefix(lower, "di isi") || strings.HasPrefix(lower, "cokan") {
		return false
	}

	// Match patterns like "SK 200", "SK200", "330.01", "01/330"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^SK\s*\d`),
		regexp.MustCompile(`(?i)^CAT\s`),
		regexp.MustCompile(`(?i)^KOB\s`),
		regexp.MustCompile(`(?i)^PC\d`),
		regexp.MustCompile(`(?i)^LDR`),
		regexp.MustCompile(`^\d{2,3}\.\d{2}`),                // "330.01"
		regexp.MustCompile(`^\d{2}/\d{3}`),                    // "01/330"
	}

	for _, p := range patterns {
		if p.MatchString(lower) {
			return true
		}
	}
	return false
}

// extractDate finds the date string from lines
func extractDate(lines []string) string {
	for _, line := range lines {
		lower := strings.ToLower(line)

		// Pattern: "Tgl:24/11/26" or "Tanggal: 24nov.2026" or "Tgl 24 /11/ 20026"
		if strings.HasPrefix(lower, "tgl") || strings.HasPrefix(lower, "tanggal") {
			val := line
			if idx := strings.Index(lower, ":"); idx >= 0 {
				val = line[idx+1:]
			} else {
				// Skip the label word
				parts := strings.Fields(line)
				if len(parts) > 1 {
					val = strings.Join(parts[1:], " ")
				}
			}
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// extractHM finds the hour meter value from lines
func extractHM(lines []string) string {
	for _, line := range lines {
		lower := strings.ToLower(line)

		// Pattern: "Hm:20852,7" or "Hm,:20852,7" or "Hm 7427,9" or just a standalone number on its own line
		if strings.HasPrefix(lower, "hm") {
			val := line
			// Remove "Hm" prefix and any following punctuation
			re := regexp.MustCompile(`(?i)^hm[,:.\s]*`)
			val = re.ReplaceAllString(val, "")
			val = strings.TrimSpace(val)
			// Remove trailing 'h' suffix
			val = strings.TrimRight(val, "hH ")
			return val
		}
	}

	// Fallback: look for a standalone number line (like "7427,9")
	reNum := regexp.MustCompile(`^\d{3,5}[.,]\d$`)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if reNum.MatchString(trimmed) {
			return trimmed
		}
	}

	return ""
}

// extractColokan finds the colokan (dipstick level) value in CM
func extractColokan(lines []string) (float64, string) {
	for _, line := range lines {
		lower := strings.ToLower(line)

		// Skip lines about "sesudah" or "isi solar"
		if strings.HasPrefix(lower, "sesudah") || strings.HasPrefix(lower, "isi solar") ||
			strings.HasPrefix(lower, "isi solar") {
			continue
		}

		// Pattern: "Colokan 33cm" or "Colokan..35.cm.star pagi" or "Clkn sebelum di isi 19 cm"
		// or "Cokan solar 39 cm" or "Colokan:53cm" or "Colokan, 50cm" or "Colokan.pagi.75.cm"
		// or "Colokan awal 17cm"
		if strings.Contains(lower, "colokan") || strings.Contains(lower, "clkn") ||
			strings.Contains(lower, "cokan") {
			// Extract number from this line
			val, ctx := extractCmFromLine(line)
			if val > 0 {
				return val, ctx
			}
		}

		// Pattern for compact format like "01/330/43/cm"
		reCompact := regexp.MustCompile(`(?i)\d{2}/\d{3}/(\d+)/cm`)
		if m := reCompact.FindStringSubmatch(line); m != nil {
			v, _ := strconv.ParseFloat(m[1], 64)
			return v, ""
		}
	}

	return 0, ""
}

// extractCmFromLine extracts a CM number and any context (pagi, malam, etc) from a colokan line
func extractCmFromLine(line string) (float64, string) {
	lower := strings.ToLower(line)

	// Normalize dots used as separators: "Colokan..35.cm.star pagi" -> "Colokan  35 cm star pagi"
	normalized := strings.ReplaceAll(lower, "..", " ")

	// Replace dots between words/numbers but keep decimal dots
	// Pattern: letter.number or number.letter -> add space
	reWordDot := regexp.MustCompile(`([a-z])\.(\d)`)
	normalized = reWordDot.ReplaceAllString(normalized, "$1 $2")
	reNumDot := regexp.MustCompile(`(\d)\.([a-z])`)
	normalized = reNumDot.ReplaceAllString(normalized, "$1 $2")
	// Replace remaining isolated dots
	normalized = strings.ReplaceAll(normalized, ".", " ")

	// Extract number followed by optional "cm"
	reNum := regexp.MustCompile(`(\d+)\s*cm`)
	if m := reNum.FindStringSubmatch(normalized); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		// Gather context words
		ctx := gatherContext(normalized)
		return v, ctx
	}

	// Fallback: just find any number in the line
	reFallback := regexp.MustCompile(`(\d+)`)
	matches := reFallback.FindAllString(normalized, -1)
	// Filter: take number that's reasonable for cm (1-150)
	for _, ms := range matches {
		v, _ := strconv.ParseFloat(ms, 64)
		if v >= 1 && v <= 150 {
			return v, gatherContext(normalized)
		}
	}

	return 0, ""
}

// gatherContext extracts context words like "pagi", "malam", "ga isi" from a line
func gatherContext(normalized string) string {
	keywords := []string{"pagi", "malam", "star", "ga isi", "ga isisolar", "brigdon", "loder", "sebelum", "awal"}
	var found []string
	for _, kw := range keywords {
		if strings.Contains(normalized, kw) {
			found = append(found, kw)
		}
	}
	return strings.Join(found, ", ")
}

// extractJerigen finds the number of jerrycans filled
func extractJerigen(lines []string) float64 {
	fullText := strings.ToLower(strings.Join(lines, "\n"))

	// Pattern: "10 jerigen" or "10jerigen" or "Di isi 10 jerigen"
	reJerigen := regexp.MustCompile(`(?i)(?:di\s*isi\s+|isi\s*solar\s*[:\s]*)?(\d+)\s*jerigen`)
	if m := reJerigen.FindStringSubmatch(fullText); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}

	// Also check for "isi solar:10jerigen"
	reIsi := regexp.MustCompile(`(?i)isi\s*solar\s*[:=]?\s*(\d+)\s*jerigen`)
	if m := reIsi.FindStringSubmatch(fullText); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}

	return 0
}

// extractSesudah finds the "sesudah" (after filling) CM value
func extractSesudah(lines []string) float64 {
	fullText := strings.ToLower(strings.Join(lines, "\n"))

	// 1. Look for pattern after jerigen: e.g. "10 jerigen clkn 59" or "10jerigen colokan 71cm"
	reAfterJerigen := regexp.MustCompile(`(?i)jerigen[^\d\n]*?(?:clkn|colokan|sesudah)?[^\d\n]*?(\d+)(?:\s*cm)?`)
	if m := reAfterJerigen.FindStringSubmatch(fullText); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		if v > 0 {
			return v
		}
	}

	// 2. Look for explicit "Sesudah" lines
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "sesudah") || strings.Contains(lower, "sesudah") {
			// Ignore empty template placeholder like "Sesudah:(cm)" or "Sesudah(cm)"
			rePrompt := regexp.MustCompile(`(?i)^sesudah\s*[:(]*\s*(?:cm\s*\)?)?\s*$`)
			if rePrompt.MatchString(lower) {
				continue
			}
			reNum := regexp.MustCompile(`(\d+)`)
			if m := reNum.FindStringSubmatch(line); m != nil {
				v, _ := strconv.ParseFloat(m[1], 64)
				if v > 0 {
					return v
				}
			}
		}
	}

	return 0
}

// containsGaIsi checks if the block says "ga isi solar" or similar
func containsGaIsi(block string) bool {
	lower := strings.ToLower(block)
	patterns := []string{
		"ga isi", "ga isisolar", "ga isi solar", "tidak isi", "tdk isi",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// ==================== UNIT MATCHING ====================

// stripNonAlphanumeric removes all spaces, dots, dashes, parentheses, symbols, etc., leaving only lowercase a-z and 0-9
func stripNonAlphanumeric(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	return strings.ToLower(re.ReplaceAllString(s, ""))
}

// matchUnit tries to match raw unit text to a master unit via alias or unit name
// Menggunakan normalisasi alfanumerik (menghilangkan spasi, titik, simbol) untuk memperbesar potensi kecocokan
func matchUnit(rawUnit string, units []SolarUnit) (bool, SolarUnit) {
	rawAlpha := stripNonAlphanumeric(rawUnit)
	if rawAlpha == "" {
		return false, SolarUnit{}
	}

	// 1. Exact alphanumeric match against Alias (e.g. "sk.330.02" -> "sk33002" == "sk33002")
	for _, u := range units {
		if u.Alias != "" && u.Alias != "0" {
			aliasAlpha := stripNonAlphanumeric(u.Alias)
			if aliasAlpha != "" && rawAlpha == aliasAlpha {
				return true, u
			}
		}
	}

	// 2. Exact alphanumeric match against Unit name (e.g. "cat950g02" == "cat950g02")
	for _, u := range units {
		unitAlpha := stripNonAlphanumeric(u.Unit)
		if unitAlpha != "" && rawAlpha == unitAlpha {
			return true, u
		}
	}

	// 3. Alphanumeric Contains: raw text contains alias (e.g. "sk20004bleker" contains "sk20004")
	for _, u := range units {
		if u.Alias != "" && u.Alias != "0" {
			aliasAlpha := stripNonAlphanumeric(u.Alias)
			if len(aliasAlpha) >= 3 && strings.Contains(rawAlpha, aliasAlpha) {
				return true, u
			}
		}
	}

	// 4. Alphanumeric Contains: raw text contains unit name (e.g. "cat950g02loader" contains "cat950g02")
	for _, u := range units {
		unitAlpha := stripNonAlphanumeric(u.Unit)
		if len(unitAlpha) >= 3 && strings.Contains(rawAlpha, unitAlpha) {
			return true, u
		}
	}

	// 5. Alphanumeric Reverse Contains: alias contains raw text (min 4 karakter alfanumerik)
	for _, u := range units {
		if u.Alias != "" && u.Alias != "0" {
			aliasAlpha := stripNonAlphanumeric(u.Alias)
			if len(rawAlpha) >= 4 && strings.Contains(aliasAlpha, rawAlpha) {
				return true, u
			}
		}
	}

	// 6. Special pattern matching untuk singkatan unik (misal 01/330 -> 330.01)
	for _, u := range units {
		if specialMatchUnit(rawUnit, u) {
			return true, u
		}
	}

	return false, SolarUnit{}
}

// cleanUnitText normalizes unit text for comparison
func cleanUnitText(s string) string {
	return stripNonAlphanumeric(s)
}

// fuzzyMatchUnit does a fuzzy comparison between two cleaned unit strings
func fuzzyMatchUnit(input, target string) bool {
	inputLower := strings.ToLower(input)
	targetLower := strings.ToLower(target)

	// If the cleaned versions match
	if inputLower == targetLower {
		return true
	}

	// Check if one contains the other (for cases like "sk05" matching "SK05")
	if strings.Contains(inputLower, targetLower) || strings.Contains(targetLower, inputLower) {
		return true
	}

	return false
}

// specialMatchUnit handles tricky patterns like "SK 200 (04 bleker" -> "KOB 04"
func specialMatchUnit(rawUnit string, u SolarUnit) bool {
	lower := strings.ToLower(rawUnit)

	// Pattern: "komatsu" anywhere -> PC200-8
	if strings.Contains(lower, "komatsu") && strings.Contains(strings.ToLower(u.Alias), "komatsu") {
		return true
	}

	// Pattern: "SK 200 (04" or "sk200-04" or "200 04" -> match alias SK200-04 or SK200-03
	re200 := regexp.MustCompile(`(?i)(?:sk\s*)?200\s*[\(\-]?\s*0?(\d)`)
	if m := re200.FindStringSubmatch(lower); m != nil {
		num := m[1]
		aliasLower := strings.ToLower(u.Alias)
		unitLower := strings.ToLower(u.Unit)
		if strings.Contains(aliasLower, "200") && strings.HasSuffix(aliasLower, "0"+num) {
			return true
		}
		if strings.Contains(unitLower, "kob") && strings.HasSuffix(unitLower, "0"+num) {
			return true
		}
	}

	// Pattern: "01/330" or "330.01" -> SK330-8 (alias 330.01)
	re330 := regexp.MustCompile(`(?i)(?:0?1[/.]330|330[/.]0?1)`)
	if re330.MatchString(lower) {
		if strings.Contains(strings.ToLower(u.Alias), "330.01") {
			return true
		}
	}

	return false
}

// ==================== DATE PARSING ====================

// parseDate handles multiple date formats from WA messages
func parseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("tanggal kosong")
	}

	// Clean common noise
	raw = strings.ReplaceAll(raw, "..", ".")
	raw = strings.TrimLeft(raw, ".")
	raw = strings.TrimRight(raw, ".")
	raw = strings.TrimSuffix(strings.ToLower(raw), ".malam")
	raw = strings.TrimSuffix(raw, " malam")
	raw = strings.TrimSuffix(raw, " pagi")

	// Remove excess spaces around slashes and dots
	re := regexp.MustCompile(`\s*/\s*`)
	raw = re.ReplaceAllString(raw, "/")
	re2 := regexp.MustCompile(`\s*\.\s*`)
	raw = re2.ReplaceAllString(raw, ".")

	// Map Indonesian month names
	monthMap := map[string]string{
		"jan": "01", "januari": "01",
		"feb": "02", "februari": "02",
		"mar": "03", "maret": "03",
		"apr": "04", "april": "04",
		"mei": "05", "may": "05",
		"jun": "06", "juni": "06",
		"jul": "07", "juli": "07",
		"agu": "08", "agustus": "08", "aug": "08",
		"sep": "09", "september": "09",
		"okt": "10", "oktober": "10", "oct": "10",
		"nov": "11", "november": "11",
		"des": "12", "desember": "12", "dec": "12",
	}

	// Try pattern: "24nov.2026" or "24.nov.2026" or "24 nov 2026" or "24nov2026" or "26 NOV 026"
	reMonthName := regexp.MustCompile(`(?i)(\d{1,2})\s*[.\s\-/]?\s*(jan(?:uari)?|feb(?:ruari)?|mar(?:et)?|apr(?:il)?|mei|may|jun(?:i)?|jul(?:i)?|agu(?:stus)?|aug|sep(?:tember)?|okt(?:ober)?|oct|nov(?:ember)?|des(?:ember)?|dec)\s*[.\s\-/]?\s*(\d{2,5})`)
	if m := reMonthName.FindStringSubmatch(raw); m != nil {
		day, _ := strconv.Atoi(m[1])
		monthStr := strings.ToLower(m[2])
		month := monthMap[monthStr]
		year := normalizeYear(m[3])
		dateStr := fmt.Sprintf("%04d-%s-%02d", year, month, day)
		return time.Parse("2006-01-02", dateStr)
	}

	// Try pattern: "dd/mm/yy" or "dd-mm-yyyy" or "dd/mm/026"
	reSlash := regexp.MustCompile(`(\d{1,2})[/\-](\d{1,2})[/\-](\d{2,5})`)
	if m := reSlash.FindStringSubmatch(raw); m != nil {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		year := normalizeYear(m[3])
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		return time.Parse("2006-01-02", dateStr)
	}

	// Try pattern: YYYY-MM-DD
	reISO := regexp.MustCompile(`(\d{4})[/\-](\d{1,2})[/\-](\d{1,2})`)
	if m := reISO.FindStringSubmatch(raw); m != nil {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		return time.Parse("2006-01-02", dateStr)
	}

	return time.Time{}, fmt.Errorf("format tanggal tidak dikenali: %s", raw)
}

// normalizeYear handles typos like "20026" -> 2026, "26" -> 2026
func normalizeYear(s string) int {
	y, _ := strconv.Atoi(s)
	if y > 2099 {
		// Typo like "20026" -> take century (2000) + 2-digit year (y % 100)
		y = 2000 + (y % 100)
	} else if y < 100 {
		y += 2000
	}
	return y
}

// ==================== NUMBER PARSING ====================

// parseNumber cleans and parses a number string with comma/dot ambiguity
func parseNumber(s string) float64 {
	if s == "" {
		return 0
	}
	// Remove spaces, trailing 'h', and leading zeros (but keep at least one digit before decimal)
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "hH ")

	// Replace comma with dot for decimal
	s = strings.ReplaceAll(s, ",", ".")

	v, _ := strconv.ParseFloat(s, 64)
	return math.Round(v*10) / 10 // round to 1 decimal
}

// ==================== DATABASE INSERT ====================

// SaveParsedReports inserts parsed reports into dt_solar_autowa
func SaveParsedReports(db *gorm.DB, reports []ParsedSolarReport) (int, []string) {
	inserted := 0
	var errors []string

	for i, r := range reports {
		jam := guessJam(r.Tanggal, r.InWa)

		err := db.Exec(`
			INSERT INTO dt_solar_autowa (tgl, jam, unit, raw_unit, hm, i_awal, i_isi, i_akhir, awal, isi, akhir, in_wa, jns_dt)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.Tanggal.Format("2006-01-02"),
			jam,
			r.Unit,
			r.RawUnit,
			r.Hm,
			r.IAwal,
			r.IIsi,
			r.IAkhir,
			r.Awal,
			r.Isi,
			r.Akhir,
			r.InWa,
			r.JenisDt,
		).Error

		if err != nil {
			errors = append(errors, fmt.Sprintf("Row #%d (%s): %v", i+1, r.Unit, err))
		} else {
			inserted++
		}
	}

	return inserted, errors
}

// guessJam determines the jam (hour) value - uses WA timestamp if available, otherwise 0
func guessJam(tgl time.Time, waTimestamp *time.Time) int {
	if waTimestamp != nil {
		return waTimestamp.Hour()
	}
	return 0
}

// ==================== WA RESPONSE FORMATTER ====================

// FormatParseResult creates a WhatsApp-friendly summary of the parse result
func FormatParseResult(result ParseResult, insertCount int, insertErrors []string) string {
	var sb strings.Builder

	if len(result.Reports) == 0 && len(result.Errors) > 0 {
		sb.WriteString("❌ *GAGAL PARSE LAPORAN SOLAR*\n")
		for _, e := range result.Errors {
			sb.WriteString(fmt.Sprintf("• %s\n", e))
		}
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("📋 *LAPORAN SOLAR - %d DATA TERPARSE*\n", len(result.Reports)))
	sb.WriteString("==============================\n")

	for i, r := range result.Reports {
		jenis := "⛽ ISI"
		if r.JenisDt == "trigger" {
			jenis = "📏 TRIGGER"
		}
		sb.WriteString(fmt.Sprintf("%d. %s | *%s*\n", i+1, jenis, r.Unit))
		sb.WriteString(fmt.Sprintf("   Tgl: %s | HM: %.1f\n", r.Tanggal.Format("02/01/2006"), r.Hm))

		if r.JenisDt == "isi" {
			sb.WriteString(fmt.Sprintf("   Awal: %s (%.1f L)\n", r.IAwal, r.Awal))
			sb.WriteString(fmt.Sprintf("   Isi: %s (%.1f L)\n", r.IIsi, r.Isi))
			if r.IAkhir != "" {
				sb.WriteString(fmt.Sprintf("   Akhir: %s (%.1f L)\n", r.IAkhir, r.Akhir))
			}
		} else {
			sb.WriteString(fmt.Sprintf("   Saldo: %s (%.1f L)\n", r.IAwal, r.Awal))
		}
		sb.WriteString("------------------------------\n")
	}

	if insertCount > 0 {
		sb.WriteString(fmt.Sprintf("\n✅ %d data berhasil disimpan ke database\n", insertCount))
	}

	if len(result.Errors) > 0 {
		sb.WriteString("\n⚠️ *Peringatan Parse:*\n")
		for _, e := range result.Errors {
			sb.WriteString(fmt.Sprintf("• %s\n", e))
		}
	}

	if len(insertErrors) > 0 {
		sb.WriteString("\n❌ *Gagal Simpan:*\n")
		for _, e := range insertErrors {
			sb.WriteString(fmt.Sprintf("• %s\n", e))
		}
	}

	return sb.String()
}

// ==================== COMPACT FORMAT PARSER ====================

// parseCompactBlock handles the ultra-compact format: "01/330/43/cm\nHM:14942, 0"
func parseCompactBlock(block string, units []SolarUnit) (ParsedSolarReport, bool) {
	report := ParsedSolarReport{}

	// Pattern: XX/XXX/NN/cm
	reCompact := regexp.MustCompile(`(?i)(\d{2})/(\d{3})/(\d+)/cm`)
	if m := reCompact.FindStringSubmatch(block); m == nil {
		return report, false
	} else {
		unitCode := m[1] + "/" + m[2] // e.g. "01/330"
		cm, _ := strconv.ParseFloat(m[3], 64)

		// Try to match unit
		matched, matchedUnit := matchUnit(unitCode, units)
		if !matched {
			return report, false
		}

		report.Unit = matchedUnit.Unit
		report.RawUnit = unitCode
		report.Konsumsi = matchedUnit.Konsumsi
		report.JenisDt = "trigger"
		report.IAwal = fmt.Sprintf("%.0f cm", cm)
		report.Awal = cm * matchedUnit.Konsumsi

		// Try to find HM
		reHm := regexp.MustCompile(`(?i)hm\s*[:.]?\s*([\d.,]+)`)
		if hm := reHm.FindStringSubmatch(block); hm != nil {
			report.Hm = parseNumber(hm[1])
		}

		return report, true
	}
}


// ==================== BOT INTEGRATION ====================

// isSolarReport detects if the incoming WA message is a solar report (not a slash command)
func isSolarReport(body string) bool {
	lower := strings.ToLower(body)
	// Must contain "laporan solar" and NOT start with "/"
	if strings.HasPrefix(strings.TrimSpace(lower), "/") {
		return false
	}
	return strings.Contains(lower, "laporan solar")
}

// GetTargetSolarGroup retrieves the configured group ID from wa_config or .env
func GetTargetSolarGroup(db *gorm.DB) string {
	if db != nil {
		var val string
		err := db.Raw("SELECT conf_value FROM wa_config WHERE conf_key = 'wa_solar_group_id' LIMIT 1").Scan(&val).Error
		if err == nil && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	// Fallback to environment variable
	return ""
}

// handleSolarReport processes an incoming solar report message from WhatsApp
func (b *WhatsAppBot) handleSolarReport(chatJID types.JID, body, senderPhone string, waTimestamp *time.Time) string {
	if b.dbCsv == nil {
		return "❌ Database tidak terhubung."
	}

	// Filter grup target jika sudah dikonfigurasi
	targetGroup := GetTargetSolarGroup(b.dbCsv)
	if targetGroup != "" {
		if chatJID.String() != targetGroup {
			// Silent ignore: bukan dari grup solar resmi
			fmt.Printf("[WhatsApp Solar] Laporan diabaikan karena bukan dari grup solar resmi (Chat: %s, Target: %s)\n",
				chatJID.String(), targetGroup)
			return ""
		}
	}

	// Parse the WA text
	result := ParseSolarWA(b.dbCsv, body, waTimestamp)

	if len(result.Reports) == 0 {
		if len(result.Errors) > 0 {
			return FormatParseResult(result, 0, nil)
		}
		return "" // empty, no reports found — don't reply
	}

	// Save to database
	insertCount, insertErrors := SaveParsedReports(b.dbCsv, result.Reports)

	// Format and return the response
	return FormatParseResult(result, insertCount, insertErrors)
}
