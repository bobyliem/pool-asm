// Modul Component Sidebar Navigation (<app-sidebar></app-sidebar>)
// 100% DINAMIS: Mengambil data menu dari database (table menu, menumaster, menu_access)
class AppSidebar extends HTMLElement {
  connectedCallback() {
    this.className = 'flex flex-col h-full max-h-full shrink-0';
    this.style.height = '100%';
    this.style.maxHeight = '100%';
    this.style.display = 'flex';

    this._onAuthUpdated = () => {
      this.render();
    };
    window.addEventListener('auth-updated', this._onAuthUpdated);

    // Jika menus belum tersimpan di sessionStorage, fetch dari backend API
    this.ensureMenusLoaded();

    this.render();
  }

  disconnectedCallback() {
    if (this._onAuthUpdated) {
      window.removeEventListener('auth-updated', this._onAuthUpdated);
    }
  }

  async ensureMenusLoaded() {
    try {
      const cu = JSON.parse(sessionStorage.getItem('current_user') || '{}');
      if (Array.isArray(cu.menus) && cu.menus.length > 0) {
        return; // Sudah ada di cache session
      }
    } catch (e) {}

    if (this._fetchingMenus) return;
    this._fetchingMenus = true;

    try {
      const res = await fetch('/api/v1/auth/user-menus');
      if (res.ok) {
        const json = await res.json();
        if (json.status === 'success' && Array.isArray(json.data)) {
          let cu = {};
          try {
            cu = JSON.parse(sessionStorage.getItem('current_user') || '{}');
          } catch (e) {}
          cu.menus = json.data;
          sessionStorage.setItem('current_user', JSON.stringify(cu));
          this.render();
        }
      }
    } catch (err) {
      console.warn('Failed to fetch dynamic menus:', err);
    } finally {
      this._fetchingMenus = false;
    }
  }

  render() {
    const activeMenu = (this.getAttribute('active-menu') || 'Dashboard').trim();
    
    // Tentukan konteks project saat ini (PRJ-HO vs PRJ-CSV)
    const path = window.location.pathname;
    let basePrefix = '/prj-csv';
    let isCsv = true;
    let isHo = false;

    if (path.includes('/prj-ho')) {
      basePrefix = '/prj-ho';
      isHo = true;
      isCsv = false;
    }

    // 1. Ambil menus dinamis dari session user
    let userMenus = [];
    let allowedMenuIds = null;
    try {
      const cu = JSON.parse(sessionStorage.getItem('current_user') || '{}');
      if (Array.isArray(cu.menus) && cu.menus.length > 0) {
        userMenus = cu.menus;
      }
      if (Array.isArray(cu.allowed_menu_ids)) {
        allowedMenuIds = cu.allowed_menu_ids.map(Number);
      }
    } catch (e) {}

    let menuStructure = [];

    if (userMenus && userMenus.length > 0) {
      // BUILD STRUKTUR 100% DINAMIS DARI DATABASE
      const targetModul = isHo ? 'PRJ-HO' : 'PRJ-CSV';
      const relevantMenus = userMenus.filter(m => (m.modul || '').toUpperCase() === targetModul);

      const groupMap = new Map();
      relevantMenus.forEach(m => {
        const title = m.modul_nama || 'MENU';
        if (!groupMap.has(title)) {
          const group = { title: title, submenus: [] };
          groupMap.set(title, group);
          menuStructure.push(group);
        }

        groupMap.get(title).submenus.push({
          id: (m.menu_nama || '').toLowerCase().replace(/[^a-z0-9]+/g, '-'),
          label: m.menu_nama,
          icon: m.icon || 'fa-circle-dot',
          href: m.url || '#',
          menuId: m.menu_id
        });
      });
    }

    // Fallback jika database menus belum tersedia di cache session
    if (menuStructure.length === 0) {
      if (isCsv) {
        menuStructure = [
          {
            title: "HOME",
            submenus: [
              { id: "dashboard", label: "Dashboard", icon: "fa-chart-line", href: "/prj-csv/dashboard", menuId: 1 }
            ]
          },
          {
            title: "SPK HARIAN",
            submenus: [
              { id: "spk-per-cust", label: "SPK Per Cust", icon: "fa-file-invoice", href: "/prj-csv/spk-per-cust", menuId: 2 },
              { id: "spk-slash-cust", label: "SPK /Cust", icon: "fa-file-invoice-dollar", href: "/prj-csv/spk-slash-cust", menuId: 17 },
              { id: "summary-per-cust", label: "Summary Per Cust", icon: "fa-table-cells-large", href: "/prj-csv/summary-per-cust", menuId: 3 },
              { id: "resume-per-kategori", label: "Resume Per Kategori", icon: "fa-border-all", href: "/prj-csv/resume-per-kategori", menuId: 4 }
            ]
          },
          {
            title: "PRODUKSI",
            submenus: [
              { id: "resume-per-hari", label: "Resume Per Hari", icon: "fa-calendar-day", href: "/prj-csv/produksi/resume-per-hari", menuId: 5 },
              { id: "summary-per-driver", label: "Summary Per Driver", icon: "fa-user-tie", href: "/prj-csv/produksi/summary-per-driver", menuId: 6 },
              { id: "monitoring-ritase", label: "Monitoring Ritase", icon: "fa-table-columns", href: "/prj-csv/produksi/monitoring-ritase", menuId: 7 },
              { id: "transfer-out", label: "Transfer Out", icon: "fa-truck-arrow-right", href: "/prj-csv/produksi/transfer-out", menuId: 16 }
            ]
          },
          {
            title: "COLOKAN SOLAR",
            submenus: [
              { id: "pemakaian-solar", label: "Pemakaian Solar", icon: "fa-gas-pump", href: "/prj-csv/colokan-solar/pemakaian-solar", menuId: 8 },
              { id: "import-data", label: "Import Data", icon: "fa-file-import", href: "/prj-csv/colokan-solar/import-data", menuId: 19 },
              { id: "detail-harian", label: "Resume Harian", icon: "fa-calendar-day", href: "/prj-csv/colokan-solar/detail-harian", menuId: 20 },
              { id: "rekap-unit", label: "Summary Per Unit", icon: "fa-chart-column", href: "/prj-csv/colokan-solar/rekap-unit", menuId: 21 },
              { id: "rekonsiliasi-anomali", label: "Laporan Rekonsiliasi & Anomali", icon: "fa-shield-halved", href: "/prj-csv/colokan-solar/rekonsiliasi-anomali", menuId: 22 },
              { id: "monitoring-tangki", label: "Monitoring Stok Tangki", icon: "fa-oil-well", href: "/prj-csv/colokan-solar/monitoring-tangki", menuId: 23 }
            ]
          },
          {
            title: "MASTER DATA",
            submenus: [
              { id: "prior-unit", label: "Prior Unit", icon: "fa-truck-fast", href: "/prj-csv/prior-unit", menuId: 9 },
              { id: "closing-finance", label: "Closing Finance", icon: "fa-file-invoice-dollar", href: "/prj-csv/closing-finance", menuId: 10 }
            ]
          },
          {
            title: "ADMINISTRATOR",
            submenus: [
              { id: "master-user", label: "Master User", icon: "fa-users-gear", href: "/admin/users", menuId: 14 },
              { id: "master-group", label: "Master Group", icon: "fa-users-rectangle", href: "/admin/groups", menuId: 15 },
              { id: "cron-jobs", label: "Cron Jobs", icon: "fa-clock", href: "/admin/cron-jobs", menuId: 18 }
            ]
          }
        ];
      } else {
        menuStructure = [
          {
            title: "OPERATION",
            submenus: [
              { id: "op-dashboard", label: "Operation Dashboard", icon: "fa-gauge-high", href: "/prj-ho/operation-dashboard", menuId: 11 }
            ]
          },
          {
            title: "SPK HARIAN",
            submenus: [
              { id: "spk-per-cust", label: "SPK Per Cust", icon: "fa-file-invoice", href: "/prj-ho/spk-per-cust", menuId: 12 }
            ]
          },
          {
            title: "WAREHOUSE",
            submenus: [
              { id: "critical-inventory", label: "Critical Inventory", icon: "fa-boxes-stacked", href: "/prj-ho/warehouse/critical-inventory", menuId: 13 }
            ]
          }
        ];
      }

      // Filter fallback berdasarkan allowedMenuIds
      menuStructure = menuStructure.map(menu => {
        const submenus = menu.submenus.filter(sub => {
          if (!allowedMenuIds) return true;
          if (sub.menuId !== undefined) {
            return allowedMenuIds.includes(Number(sub.menuId));
          }
          return true;
        });
        return { ...menu, submenus };
      }).filter(menu => menu.submenus.length > 0);
    }

    let menuHTML = `
      <style>
        #sidebarNav::-webkit-scrollbar {
          width: 6px;
        }
        #sidebarNav::-webkit-scrollbar-track {
          background: #f8fafc;
        }
        #sidebarNav::-webkit-scrollbar-thumb {
          background: #cbd5e1;
          border-radius: 4px;
        }
        #sidebarNav::-webkit-scrollbar-thumb:hover {
          background: #94a3b8;
        }
      </style>
      <!-- Backdrop Gelap di Layar Mobile -->
      <div id="sidebarBackdrop" onclick="window.toggleSidebar()" class="fixed inset-0 bg-slate-900/50 backdrop-blur-2xs z-30 hidden md:hidden transition-opacity"></div>

      <!-- Drawer Sidebar (Di Mobile: Offcanvas Drawer melayang dari kiri, Di Desktop: Sidebar inline normal) -->
      <aside id="sidebarNav" class="fixed inset-y-0 left-0 z-40 w-64 bg-white border-r border-gray-200 p-4 pb-12 flex flex-col space-y-4 overflow-y-auto h-full max-h-full shrink-0 shadow-2xl transition-transform duration-300 transform -translate-x-full md:static md:translate-x-0 md:shadow-none" style="scrollbar-width: thin; scrollbar-color: #cbd5e1 #f8fafc;">
        
        <!-- Mobile Drawer Header (Title + Tombol Tutup X) -->
        <div class="flex items-center justify-between pb-2 border-b border-gray-100 md:hidden">
          <div class="flex items-center gap-2">
            <img src="/web/images/logo-sar.png" onerror="this.onerror=null;this.src='https://workshop.sampurna-group.com/images/logo-sar.png'" alt="SAR Logo" class="h-6 w-auto object-contain" />
            <span class="font-bold text-xs text-gray-800">Menu Navigasi</span>
          </div>
          <button onclick="window.toggleSidebar()" class="text-gray-400 hover:text-gray-700 p-1 rounded-lg focus:outline-none">
            <i class="fa-solid fa-xmark text-base"></i>
          </button>
        </div>

        <!-- Project Switcher Pill -->
        <div class="bg-slate-100 p-1.5 rounded-lg flex items-center justify-between text-xs font-medium">
          <a href="/prj-ho" class="flex-1 text-center py-1.5 rounded-md transition ${isHo ? 'bg-white text-blue-600 font-bold shadow-sm' : 'text-gray-500 hover:text-gray-800'}">
            PRJ-HO
          </a>
          <a href="/prj-csv" class="flex-1 text-center py-1.5 rounded-md transition ${isCsv ? 'bg-white text-blue-600 font-bold shadow-sm' : 'text-gray-500 hover:text-gray-800'}">
            PRJ-CSV
          </a>
        </div>
    `;

    menuStructure.forEach(menu => {
      menuHTML += `
        <div>
          <h2 class="text-xs font-bold text-gray-400 uppercase tracking-wider mb-2 px-3">${menu.title}</h2>
          <ul class="space-y-1">
      `;

      menu.submenus.forEach(sub => {
        const isExactMatch = sub.href && sub.href !== '#' && (path === sub.href || path === sub.href + '.html');
        const isPrefixMatch = sub.href && sub.href !== '#' && sub.href !== '/prj-csv' && sub.href !== '/prj-ho' && path.startsWith(sub.href + '/');
        const isLabelMatch = sub.label.toLowerCase() === activeMenu.toLowerCase() || sub.id === activeMenu.toLowerCase();
        
        const isActive = isExactMatch || isPrefixMatch || isLabelMatch;
        
        const activeClass = isActive 
          ? "text-blue-600 font-semibold bg-blue-50 border-r-4 border-blue-600" 
          : "text-gray-600 hover:bg-gray-100 hover:text-gray-900";

        menuHTML += `
          <li>
            <a href="${sub.href}" 
               data-menu-id="${sub.id}" onclick="if(window.innerWidth < 768) window.toggleSidebar();"
               class="flex items-center space-x-3 px-3 py-2 rounded-lg text-sm transition duration-150 ${activeClass}">
              <i class="fa-solid ${sub.icon} w-4 text-center"></i>
              <span>${sub.label}</span>
            </a>
          </li>
        `;
      });

      menuHTML += `
          </ul>
        </div>
      `;
    });

    menuHTML += `</aside>`;
    this.innerHTML = menuHTML;

    // Otomatis scroll sidebar agar submenu yang aktif langsung terlihat di tengah layar
    this.scrollToActiveMenu();
  }

  scrollToActiveMenu() {
    const doScroll = () => {
      const aside = this.querySelector('#sidebarNav');
      const activeLink = this.querySelector('a.border-blue-600') || this.querySelector('a.text-blue-600');
      if (aside && activeLink) {
        const asideRect = aside.getBoundingClientRect();
        const linkRect = activeLink.getBoundingClientRect();
        
        // Cek jika link berada di luar area tengah yang nyaman dilihat (jarak 30px dari atas/bawah)
        if (linkRect.bottom > asideRect.bottom - 30 || linkRect.top < asideRect.top + 30) {
          activeLink.scrollIntoView({ block: 'center', behavior: 'auto' });
        }
      }
    };

    requestAnimationFrame(doScroll);
    setTimeout(doScroll, 100);
    setTimeout(doScroll, 300);
  }
}

if (!customElements.get('app-sidebar')) {
  customElements.define('app-sidebar', AppSidebar);
}
