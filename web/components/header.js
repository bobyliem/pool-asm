// Modul Component Header (<app-header></app-header>)
class AppHeader extends HTMLElement {
  connectedCallback() {
    // Set Favicon otomatis dari Logo Header
    let favicon = document.querySelector("link[rel*='icon']");
    if (!favicon) {
      favicon = document.createElement('link');
      favicon.rel = 'icon';
      favicon.type = 'image/png';
      document.head.appendChild(favicon);
    }
    favicon.href = '/web/images/logo-sar.png';

    let initialName = 'Memuat...';
    let initialGroup = '-';
    try {
      const cached = JSON.parse(sessionStorage.getItem('current_user') || '{}');
      if (cached && (cached.nama || cached.username)) {
        initialName = cached.nama || cached.username;
        initialGroup = cached.group_name || '-';
      }
    } catch(e) {}
    const userName = initialName;

    const appTitle = this.getAttribute('app-title') || 'Sampurna Workshop';
    
    // Cek apakah halaman berada di konteks PRJ-CSV atau PRJ-HO
    const isCsv = window.location.pathname.includes('/prj-csv');

    this.innerHTML = `
      <header class="bg-white border-b border-gray-200 px-3 sm:px-6 py-2.5 sm:py-3 flex items-center justify-between shadow-xs sticky top-0 z-30">
        <div class="flex items-center gap-2 sm:gap-3">
          <!-- Tombol Hamburger: Paling kiri sebelum logo agar mudah dijangkau jempol -->
          <button onclick="window.toggleSidebar()" class="text-gray-600 hover:text-gray-900 p-1.5 -ml-1 rounded-lg hover:bg-gray-100 focus:outline-none cursor-pointer" title="Menu Navigasi">
            <i class="fa-solid fa-bars text-lg"></i>
          </button>

          <!-- Logo SAR & Judul (Di mobile HANYA logo yang muncul agar space lega) -->
          <a href="/prj-csv" class="flex items-center gap-2">
            <img src="/web/images/logo-sar.png" onerror="this.onerror=null;this.src='https://workshop.sampurna-group.com/images/logo-sar.png'" alt="SAR Logo" class="h-7 sm:h-8 w-auto object-contain" />
            <h1 class="hidden sm:block text-base sm:text-lg font-bold text-gray-800 tracking-wide">${appTitle}</h1>
          </a>
        </div>

        <div class="flex items-center space-x-3">
          <!-- User Profile Dropdown Button -->
          <div class="relative">
            <div id="navUserProfileBtn" onclick="window.toggleUserMenu(event)" class="border border-blue-200/80 rounded-xl px-2.5 sm:px-3 py-1 flex items-center gap-2 sm:gap-2.5 bg-blue-50/70 hover:bg-blue-100/80 cursor-pointer transition select-none shadow-2xs">
              <div class="w-7 h-7 rounded-full bg-blue-600 text-white flex items-center justify-center font-bold text-xs shrink-0 shadow-2xs">
                <i class="fa-regular fa-user text-xs"></i>
              </div>
              <div class="flex flex-col text-left leading-tight">
                <span id="navUserName" class="text-xs font-bold text-slate-800 truncate max-w-[85px] sm:max-w-[140px]">${initialName}</span>
                <span id="navUserGroup" class="text-[9.5px] sm:text-[10px] font-semibold text-blue-600 uppercase tracking-wider truncate max-w-[85px] sm:max-w-[140px]">${initialGroup}</span>
              </div>
              <i class="fa-solid fa-chevron-down text-[10px] text-slate-400 ml-0.5"></i>
            </div>

            <!-- Dropdown Menu -->
            <div id="navUserDropdown" class="hidden absolute right-0 mt-2 w-56 bg-white border border-slate-200 rounded-xl shadow-xl py-1.5 z-50 text-xs">
              <div class="px-3.5 py-2 border-b border-slate-100 text-[11px] bg-slate-50/60 rounded-t-xl">
                <div class="font-bold text-slate-800 truncate" id="dropdownUserNama">${userName}</div>
                <div class="text-blue-600 text-[10px] font-semibold uppercase tracking-wider truncate" id="dropdownUserGroup">${initialGroup}</div>
              </div>

              <!-- Tombol Sync CSV: Ditaruh di bawah nama user di kanan atas! Hemat space & rapi! -->
              ${isCsv ? `
                <button id="navBtnSyncCsv" onclick="window.triggerNavCsvSync(); window.closeUserMenu();" class="w-full text-left px-3.5 py-2.5 text-blue-600 hover:bg-blue-50 font-semibold flex items-center gap-2.5 transition cursor-pointer border-b border-slate-100" title="Sinkronisasi Data CSV Terbaru dari Background">
                  <i id="navBtnSyncIcon" class="fa-solid fa-rotate text-xs text-blue-500"></i>
                  <span id="navBtnSyncText">Sync CSV</span>
                </button>
              ` : ''}

              <button onclick="window.openChangePasswordModal()" class="w-full text-left px-3.5 py-2.5 text-slate-700 hover:bg-blue-50 hover:text-blue-600 flex items-center gap-2.5 transition cursor-pointer">
                <i class="fa-solid fa-key text-xs text-blue-500"></i>
                <span>Change Pass</span>
              </button>
              <button onclick="window.handleUserLogout()" class="w-full text-left px-3.5 py-2.5 text-rose-600 hover:bg-rose-50 flex items-center gap-2.5 transition cursor-pointer border-t border-slate-100">
                <i class="fa-solid fa-right-from-bracket text-xs text-rose-500"></i>
                <span>Logout</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      <!-- Banner Status Sinkronisasi di Navbar (Hanya untuk PRJ-CSV) -->
      ${isCsv ? `
        <div id="navSyncBanner" class="hidden transition-all duration-300 border-b px-6 py-2 text-xs flex items-center justify-between shadow-2xs bg-blue-50 border-blue-200 text-blue-900 sticky top-[57px] z-20">
          <div class="flex items-center gap-2.5">
            <i id="navSyncBannerIcon" class="fa-solid fa-circle-notch fa-spin text-blue-600 text-sm"></i>
            <div>
              <span id="navSyncBannerTitle" class="font-bold">Menyinkronkan Data CSV...</span>
              <span id="navSyncBannerMsg" class="text-slate-600 text-[11px] ml-2">Sedang mengimpor data ke MySQL...</span>
            </div>
          </div>
          <button onclick="window.dismissNavSyncBanner()" class="text-slate-400 hover:text-slate-600 text-xs px-2 py-1 cursor-pointer">
            <i class="fa-solid fa-xmark"></i>
          </button>
        </div>
      ` : ''}

      <!-- GLOBAL MODAL CHANGE PASSWORD -->
      <div id="globalModalChangePass" class="fixed inset-0 bg-slate-900/50 backdrop-blur-2xs z-50 flex items-center justify-center hidden">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 border border-slate-200 overflow-hidden text-xs">
          <div class="px-4 py-3 bg-slate-800 text-white flex items-center justify-between">
            <div class="flex items-center gap-2">
              <i class="fa-solid fa-key text-blue-400 text-sm"></i>
              <h3 class="font-bold text-sm">Ubah Password</h3>
            </div>
            <button onclick="window.closeChangePasswordModal()" class="text-slate-400 hover:text-white cursor-pointer">
              <i class="fa-solid fa-xmark text-base"></i>
            </button>
          </div>

          <form onsubmit="window.submitChangePassword(event)" class="p-4 space-y-3">
            <div>
              <label class="block text-[11px] font-bold text-slate-700 mb-1">Password Lama <span class="text-red-500">*</span></label>
              <input type="password" id="cpOldPass" required placeholder="Masukkan password lama" class="w-full px-2.5 py-1.5 text-xs border border-slate-300 rounded focus:ring-1 focus:ring-blue-500 focus:outline-none" />
            </div>

            <div>
              <label class="block text-[11px] font-bold text-slate-700 mb-1">Password Baru <span class="text-red-500">*</span></label>
              <input type="password" id="cpNewPass" required minlength="4" placeholder="Minimal 4 karakter" class="w-full px-2.5 py-1.5 text-xs border border-slate-300 rounded focus:ring-1 focus:ring-blue-500 focus:outline-none" />
            </div>

            <div>
              <label class="block text-[11px] font-bold text-slate-700 mb-1">Konfirmasi Password Baru <span class="text-red-500">*</span></label>
              <input type="password" id="cpConfirmPass" required minlength="4" placeholder="Ulangi password baru" class="w-full px-2.5 py-1.5 text-xs border border-slate-300 rounded focus:ring-1 focus:ring-blue-500 focus:outline-none" />
            </div>

            <div id="cpAlert" class="hidden p-2 rounded text-[11px] bg-rose-50 text-rose-800 border border-rose-200"></div>

            <div class="pt-3 border-t border-slate-200 flex items-center justify-end gap-2">
              <button type="button" onclick="window.closeChangePasswordModal()" class="px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded text-xs">Batal</button>
              <button type="submit" id="btnSubmitChangePass" class="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded text-xs shadow-2xs">Simpan Password</button>
            </div>
          </form>
        </div>
      </div>
    `;

    // Tutup dropdown jika klik di luar
    document.addEventListener('click', (e) => {
      const btn = document.getElementById('navUserProfileBtn');
      const dropdown = document.getElementById('navUserDropdown');
      if (btn && dropdown && !btn.contains(e.target) && !dropdown.contains(e.target)) {
        dropdown.classList.add('hidden');
      }
    });

    // Validasi session dan sinkronisasi
    window.checkAuthSession();

    if (isCsv) {
      window.checkNavSyncStatus();
    }
  }
}

// ==========================================
// USER AUTHENTICATION & SESSION MANAGEMENT
// ==========================================
window.checkAuthSession = async function() {
  const currentPath = window.location.pathname;
  if (currentPath === '/login' || currentPath.endsWith('login.html')) {
    return;
  }

  // Cek cache session agar tampilan langsung muncul instan tanpa flicker
  const cachedUserStr = sessionStorage.getItem('current_user');
  if (cachedUserStr) {
    try {
      const cu = JSON.parse(cachedUserStr);
      const nameEl = document.getElementById('navUserName');
      const groupEl = document.getElementById('navUserGroup');
      const dropName = document.getElementById('dropdownUserNama');
      const dropGroup = document.getElementById('dropdownUserGroup');

      if (nameEl) nameEl.textContent = cu.nama || cu.username;
      if (groupEl) groupEl.textContent = cu.group_name || ('Group ' + cu.grp_id);
      if (dropName) dropName.textContent = cu.nama || cu.username;
      if (dropGroup) dropGroup.textContent = cu.group_name || ('Group ' + cu.grp_id);
    } catch(e) {}
  }

  try {
    const res = await fetch('/api/v1/auth/me');
    if (!res.ok) {
      sessionStorage.removeItem('current_user');
      window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname + window.location.search);
      return;
    }
    const json = await res.json();
    if (json.status === 'success' && json.data) {
      const u = json.data;
      sessionStorage.setItem('current_user', JSON.stringify(u));

      const nameEl = document.getElementById('navUserName');
      const groupEl = document.getElementById('navUserGroup');
      const dropName = document.getElementById('dropdownUserNama');
      const dropGroup = document.getElementById('dropdownUserGroup');

      if (nameEl) nameEl.textContent = u.nama || u.username;
      if (groupEl) groupEl.textContent = u.group_name || ('Group ' + u.grp_id);
      if (dropName) dropName.textContent = u.nama || u.username;
      if (dropGroup) dropGroup.textContent = u.group_name || ('Group ' + u.grp_id);
    }
  } catch (err) {
    window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname + window.location.search);
  }
};

window.toggleUserMenu = function(e) {
  e.stopPropagation();
  const dropdown = document.getElementById('navUserDropdown');
  if (dropdown) {
    dropdown.classList.toggle('hidden');
  }
};

window.closeUserMenu = function() {
  const dropdown = document.getElementById('navUserDropdown');
  if (dropdown) {
    dropdown.classList.add('hidden');
  }
};

window.handleUserLogout = async function() {
  if (!confirm('Apakah Anda yakin ingin logout?')) {
    return;
  }
  try {
    await fetch('/api/v1/auth/logout', { method: 'POST' });
  } catch (e) {
    // ignore
  }
  sessionStorage.removeItem('current_user');
  window.location.href = '/login';
};

window.openChangePasswordModal = function() {
  const dropdown = document.getElementById('navUserDropdown');
  if (dropdown) dropdown.classList.add('hidden');

  document.getElementById('cpOldPass').value = '';
  document.getElementById('cpNewPass').value = '';
  document.getElementById('cpConfirmPass').value = '';
  const alertEl = document.getElementById('cpAlert');
  if (alertEl) alertEl.classList.add('hidden');

  const modal = document.getElementById('globalModalChangePass');
  if (modal) modal.classList.remove('hidden');
};

window.closeChangePasswordModal = function() {
  const modal = document.getElementById('globalModalChangePass');
  if (modal) modal.classList.add('hidden');
};

window.submitChangePassword = async function(e) {
  e.preventDefault();
  const oldPass = document.getElementById('cpOldPass').value;
  const newPass = document.getElementById('cpNewPass').value;
  const confirmPass = document.getElementById('cpConfirmPass').value;
  const alertEl = document.getElementById('cpAlert');

  if (newPass !== confirmPass) {
    alertEl.textContent = 'Konfirmasi password baru tidak cocok!';
    alertEl.classList.remove('hidden');
    return;
  }

  const btn = document.getElementById('btnSubmitChangePass');
  btn.disabled = true;

  try {
    const res = await fetch('/api/v1/auth/change-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ old_password: oldPass, new_password: newPass })
    });
    const result = await res.json();
    if (res.ok && result.status === 'success') {
      alert('Password berhasil diubah!');
      window.closeChangePasswordModal();
    } else {
      alertEl.textContent = result.message || 'Gagal mengubah password.';
      alertEl.classList.remove('hidden');
    }
  } catch (err) {
    alertEl.textContent = 'Kesalahan jaringan: ' + err.message;
    alertEl.classList.remove('hidden');
  } finally {
    btn.disabled = false;
  }
};

window.toggleSidebar = function() {
  const sidebar = document.getElementById('sidebarNav');
  const backdrop = document.getElementById('sidebarBackdrop');
  if (!sidebar) return;

  const isMobile = window.innerWidth < 768;
  if (isMobile) {
    const isClosed = sidebar.classList.contains('-translate-x-full');
    if (isClosed) {
      sidebar.classList.remove('-translate-x-full');
      sidebar.classList.add('translate-x-0');
      if (backdrop) backdrop.classList.remove('hidden');
    } else {
      sidebar.classList.add('-translate-x-full');
      sidebar.classList.remove('translate-x-0');
      if (backdrop) backdrop.classList.add('hidden');
    }
  } else {
    sidebar.classList.toggle('md:hidden');
  }
};

// ==========================================
// UNIFIED DATE SESSION MANAGER (ASM_SESSION)
// ==========================================
window.ASM_SESSION = {
  getTodayDate: function() {
    const d = new Date();
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${y}-${m}-${day}`;
  },

  // Ambil periode dari session. Jika tidak ada, default hari ini & simpan ke session.
  getDateRange: function() {
    const today = this.getTodayDate();
    let tglAwal = sessionStorage.getItem('asm_tgl_awal') || sessionStorage.getItem('csv_spk_tglawal');
    let tglAkhir = sessionStorage.getItem('asm_tgl_akhir') || sessionStorage.getItem('csv_spk_tglakhir');

    if (!tglAwal) {
      tglAwal = today;
      sessionStorage.setItem('asm_tgl_awal', tglAwal);
      sessionStorage.setItem('csv_spk_tglawal', tglAwal);
    }
    if (!tglAkhir) {
      tglAkhir = today;
      sessionStorage.setItem('asm_tgl_akhir', tglAkhir);
      sessionStorage.setItem('csv_spk_tglakhir', tglAkhir);
    }

    return { tglAwal, tglAkhir };
  },

  // Simpan periode baru ke session
  setDateRange: function(tglAwal, tglAkhir) {
    const today = this.getTodayDate();
    if (!tglAwal) tglAwal = today;
    if (!tglAkhir) tglAkhir = today;

    sessionStorage.setItem('asm_tgl_awal', tglAwal);
    sessionStorage.setItem('asm_tgl_akhir', tglAkhir);
    sessionStorage.setItem('csv_spk_tglawal', tglAwal);
    sessionStorage.setItem('csv_spk_tglakhir', tglAkhir);

    return { tglAwal, tglAkhir };
  },

  // Helper untuk mengisi elemen input date dari session
  initDateInputs: function(elemIdAwal = 'filterTglAwal', elemIdAkhir = 'filterTglAkhir') {
    const { tglAwal, tglAkhir } = this.getDateRange();
    const elAwal = document.getElementById(elemIdAwal);
    const elAkhir = document.getElementById(elemIdAkhir);
    if (elAwal) elAwal.value = tglAwal;
    if (elAkhir) elAkhir.value = tglAkhir;
    return { tglAwal, tglAkhir };
  }
};

// ==========================================
// GLOBAL NAVBAR CSV SYNC MANAGER
// ==========================================
let navSyncPollTimer = null;

window.triggerNavCsvSync = async function() {
  const btn = document.getElementById('navBtnSyncCsv');
  const icon = document.getElementById('navBtnSyncIcon');
  const text = document.getElementById('navBtnSyncText');
  
  if (btn) btn.disabled = true;
  if (icon) icon.className = 'fa-solid fa-spinner fa-spin text-blue-600';
  if (text) text.innerText = 'Menyinkronkan...';

  window.showNavSyncBanner('running', 'Memulai Sinkronisasi CSV...', 'Mengirim sinyal background import ke server...');

  try {
    const res = await fetch('/api/v1/csv/sync', { 
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    const data = await res.json();

    if (!res.ok || data.status === 'busy' || data.status === 'error') {
      window.resetNavSyncBtn();
      window.dismissNavSyncBanner();
      alert(data.message || 'Last update less than 5 minutes, try again later.');
      return;
    }

    window.startNavSyncPolling();
  } catch (err) {
    console.error('Trigger sync error:', err);
    window.resetNavSyncBtn();
    window.showNavSyncBanner('error', 'Gagal Memulai Sinkronisasi', err.message);
  }
};

window.checkNavSyncStatus = async function() {
  try {
    const res = await fetch('/api/v1/csv/sync/status');
    const json = await res.json();
    if (json && json.data && json.data.is_running) {
      const btn = document.getElementById('navBtnSyncCsv');
      const icon = document.getElementById('navBtnSyncIcon');
      const text = document.getElementById('navBtnSyncText');
      if (btn) btn.disabled = true;
      if (icon) icon.className = 'fa-solid fa-spinner fa-spin text-blue-600';
      if (text) text.innerText = 'Menyinkronkan...';
      window.showNavSyncBanner('running', 'Menyinkronkan Data CSV...', json.data.message || 'Sedang mengimpor data...');
      window.startNavSyncPolling();
    }
  } catch (e) {
    // ignore
  }
};

window.startNavSyncPolling = function() {
  if (navSyncPollTimer) clearInterval(navSyncPollTimer);
  navSyncPollTimer = setInterval(async () => {
    try {
      const res = await fetch('/api/v1/csv/sync/status');
      const json = await res.json();
      if (json && json.data) {
        const st = json.data;
        if (st.is_running) {
          window.showNavSyncBanner('running', 'Menyinkronkan Data CSV...', st.message || 'Sedang mengimpor data ke MySQL...');
        } else {
          clearInterval(navSyncPollTimer);
          navSyncPollTimer = null;
          window.resetNavSyncBtn();

          if (st.status === 'success') {
            window.showNavSyncBanner('success', 'Sinkronisasi Selesai!', 'Data CSV terbaru berhasil diimpor. Memperbarui halaman...');
            
            if (typeof window.fetchDashboardCategoryData === 'function') {
              window.fetchDashboardCategoryData();
            }
            if (typeof window.fetchSpkData === 'function') {
              if (typeof window.loadFilterOptions === 'function') window.loadFilterOptions();
              window.fetchSpkData();
            }
            window.dispatchEvent(new CustomEvent('csv-sync-finished'));

            setTimeout(() => window.dismissNavSyncBanner(), 4000);
          } else if (st.status === 'error') {
            window.showNavSyncBanner('error', 'Sinkronisasi Mengalami Kendala', st.message);
          }
        }
      }
    } catch (e) {
      console.error('Poll status error:', e);
    }
  }, 1500);
};

window.resetNavSyncBtn = function() {
  const btn = document.getElementById('navBtnSyncCsv');
  const icon = document.getElementById('navBtnSyncIcon');
  const text = document.getElementById('navBtnSyncText');
  if (btn) btn.disabled = false;
  if (icon) icon.className = 'fa-solid fa-rotate';
  if (text) text.innerText = 'Sync CSV';
};

window.showNavSyncBanner = function(type, title, msg) {
  const banner = document.getElementById('navSyncBanner');
  const icon = document.getElementById('navSyncBannerIcon');
  const titleEl = document.getElementById('navSyncBannerTitle');
  const msgEl = document.getElementById('navSyncBannerMsg');
  if (!banner) return;

  banner.classList.remove('hidden', 'bg-blue-50', 'border-blue-200', 'text-blue-900', 'bg-emerald-50', 'border-emerald-200', 'text-emerald-900', 'bg-rose-50', 'border-rose-200', 'text-rose-900');

  if (type === 'running') {
    banner.classList.add('bg-blue-50', 'border-b', 'border-blue-200', 'text-blue-900');
    if (icon) icon.className = 'fa-solid fa-circle-notch fa-spin text-blue-600 text-sm';
  } else if (type === 'success') {
    banner.classList.add('bg-emerald-50', 'border-b', 'border-emerald-200', 'text-emerald-900');
    if (icon) icon.className = 'fa-solid fa-circle-check text-emerald-600 text-sm';
  } else {
    banner.classList.add('bg-rose-50', 'border-b', 'border-rose-200', 'text-rose-900');
    if (icon) icon.className = 'fa-solid fa-circle-exclamation text-rose-600 text-sm';
  }

  if (titleEl) titleEl.innerText = title;
  if (msgEl) msgEl.innerText = msg;
};

window.dismissNavSyncBanner = function() {
  const banner = document.getElementById('navSyncBanner');
  if (banner) banner.classList.add('hidden');
};

if (!customElements.get('app-header')) {
  customElements.define('app-header', AppHeader);
}
