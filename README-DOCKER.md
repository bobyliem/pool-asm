# Panduan Deployment Docker - ASM (Aplikasi Service & Monitoring)

Dokumen ini adalah panduan lengkap untuk tim implementor/DevOps dalam melakukan deployment aplikasi **ASM (Golang)** beserta **MySQL Database** menggunakan Docker Compose.

---

## 1. Arsitektur & Topologi

Stack Docker mengemas aplikasi dan database lokal dalam satu jaringan internal (`asm-net`):

```
+--------------------------------------------------------------+
| Host Machine                                                 |
|                                                              |
|   [Port 8080] -----------> Container: asm-app (Golang App)   |
|                                 |                            |
|   [Port 3307] -----------> Container: asm-mysql-csv          |
|                                 (Database: imp_sampurna)     |
|                                 (Volume: mysql_data)         |
|                                                              |
|   Container asm-app juga terhubung ke:                       |
|   -> Database Head Office (HO) di IP 10.2.1.2:3306 (Remote)  |
+--------------------------------------------------------------+
```

| Komponen | Container Name | Port Host | Port Container | Keterangan |
| :--- | :--- | :--- | :--- | :--- |
| **Aplikasi ASM** | `asm-app` | `8080` | `8080` | Golang Alpine (Web & REST API) |
| **MySQL CSV** | `asm-mysql-csv` | `3307` | `3306` | Database lokal `imp_sampurna` + auto-init script |
| **MySQL HO** | *(Server Remote)* | - | `3306` | Database HO `workshop` (IP: 10.2.1.2) |

---

## 2. Prasyarat Sistem

Pastikan di server target sudah terinstal:
- **Docker Engine** (versi 24.0+ disarankan)
- **Docker Compose** v2 (perintah `docker compose`)
- Konektivitas jaringan ke IP Database Head Office (`10.2.1.2:3306`)

---

## 3. Langkah-Langkah Deployment

### Langkah 1: Siapkan File Environment (`.env`)
Salin file `.env.example` menjadi `.env`:
```bash
cp .env.example .env
```
*(Di Windows PowerShell: `Copy-Item .env.example .env`)*

Periksa isi file `.env`:
```ini
# Port Web Server
PORT=8080

# Database Lokal CSV (MySQL Container)
DB_CSV_HOST=db-csv
DB_CSV_PORT=3306
DB_CSV_USER=root
DB_CSV_PASS=P@ssw0rd
DB_CSV_NAME=imp_sampurna

# Database Head Office (Remote HO Server)
DB_HO_HOST=10.2.1.2
DB_HO_PORT=3306
DB_HO_USER=boby
DB_HO_PASS=Boby@150926
DB_HO_NAME=workshop
```

> **Catatan Penting:** Di dalam Docker network, `DB_CSV_HOST` diisi dengan nama service yaitu `db-csv` dan port container `3306`. Jangan diubah ke `localhost` kecuali aplikasi dijalankan tanpa Docker.

---

### Langkah 2: Build & Jalankan Container
Jalankan perintah berikut di folder root aplikasi:
```bash
docker compose up -d --build
```

**Proses yang berjalan otomatis:**
1. Docker akan men-download image `mysql:8.0` dan `golang:1.24-alpine`.
2. Multi-stage build akan mengompilasi binary Golang secara statis dan mengemasnya ke runner image Alpine yang sangat ringan (< 30 MB).
3. Service `db-csv` akan start pertama kali. Pada saat container dibuat pertama kali, script dump di `docker/init-db/01-init.sql` akan otomatis dieksekusi untuk membuat seluruh tabel, view, dan trigger database `imp_sampurna`.
4. Service `app` akan menunggu sampai `db-csv` berstatus **healthy**, baru aplikasi Golang dijalankan.

---

### Langkah 3: Verifikasi Status Container
Cek apakah kedua container sudah berjalan normal:
```bash
docker compose ps
```
Output yang diharapkan:
```
NAME            IMAGE     COMMAND                  SERVICE   STATUS                    PORTS
asm-app         asm-app   "/app/asm"               app       Up (healthy)              0.0.0.0:8080->8080/tcp
asm-mysql-csv   mysql:8.0 "docker-entrypoint.s…"   db-csv    Up (healthy)              0.0.0.0:3307->3306/tcp
```

---

### Langkah 4: Cek Log Aplikasi
Untuk melihat log realtime aplikasi:
```bash
docker compose logs -f app
```

Untuk melihat log MySQL:
```bash
docker compose logs -f db-csv
```

---

## 4. Akses Aplikasi

- **Web Browser**: Buka `http://<IP_SERVER>:8080/`
- **Login Default**:
  - Username: `boby`
  - Password: `123` (atau sesuai password yang telah diatur)
- **Akses MySQL Langsung dari Host (Navicat / DBeaver / HeidiSQL)**:
  - Host: `localhost` (atau IP server)
  - Port: `3307`
  - User: `root`
  - Password: `P@ssw0rd`
  - Database: `imp_sampurna`

---

## 5. Maintenance & Operasional

### Menghentikan Container
```bash
docker compose down
```
*(Data database tetap aman karena disimpan di Docker volume `mysql_data`)*

### Menjalankan Ulang (Restart)
```bash
docker compose restart
```

### Update Kode Aplikasi (Re-deploy)
Jika ada update source code dari Git:
```bash
git pull
docker compose up -d --build app
```

### Reset / Inisialisasi Ulang Database dari Nol
> ⚠️ **PERINGATAN: Perintah ini akan menghapus seluruh data di volume MySQL.**
```bash
docker compose down -v
docker compose up -d --build
```
*(Flag `-v` akan menghapus volume `mysql_data` sehingga saat start ulang script `01-init.sql` akan dieksekusi kembali).*

---

## 6. Struktur File Docker Terkait

```
.
├── Dockerfile                  # Multi-stage build Go (Builder -> Alpine Runner)
├── .dockerignore               # File/folder yang diabaikan saat docker build
├── docker-compose.yml          # Definisi service app & db-csv
├── .env.example                # Template konfigurasi environment
├── docker/
│   └── init-db/
│       └── 01-init.sql         # Database dump auto-init (tabel, triggers, initial data)
└── README-DOCKER.md            # Panduan ini
```
