// Modul Component Sidebar Navigation (<app-sidebar></app-sidebar>)
class AppSidebar extends HTMLElement {
  connectedCallback() {
    this.className = 'flex flex-col h-full max-h-full shrink-0';
    this.style.height = '100%';
    this.style.maxHeight = '100%';
    this.style.display = 'flex';

    const activeMenu = this.getAttribute('active-menu') || 'Dashboard';
    
    // Determine current project context
    const path = window.location.pathname;
    let basePrefix = '';
    let isCsv = false;
    let isHo = false;

    if (path.includes('/prj-csv') || path.includes('/colokan-solar')) {
      basePrefix = '/prj-csv';
      isCsv = true;
    } else if (path.includes('/prj-ho')) {
      basePrefix = '/prj-ho';
      isHo = true;
    }

    let menuStructure = [];

    if (isCsv) {
      // Menu khusus PRJ-CSV
      menuStructure = [
        {
          title: "HOME",
          submenus: [
            { id: "dashboard", label: "Dashboard", icon: "fa-chart-line", href: "/prj-csv/dashboard" }
          ]
        },
        {
          title: "SPK Harian",
          submenus: [
            { id: "spk-per-cust", label: "SPK Per Cust", icon: "fa-file-invoice", href: "/prj-csv/spk-per-cust" },
            { id: "spk-slash-cust", label: "SPK /Cust", icon: "fa-file-invoice-dollar", href: "/prj-csv/spk-slash-cust" },
            { id: "summary-per-cust", label: "Summary Per Cust", icon: "fa-table-cells-large", href: "/prj-csv/summary-per-cust" },
            { id: "resume-per-kategori", label: "Resume Per Kategori", icon: "fa-border-all", href: "/prj-csv/resume-per-kategori" }
          ]
        },
        {
          title: "PRODUKSI",
          submenus: [
            { id: "resume-per-hari", label: "Resume Per Hari", icon: "fa-calendar-day", href: "/prj-csv/produksi/resume-per-hari" },
            { id: "summary-per-driver", label: "Summary Per Driver", icon: "fa-user-tie", href: "/prj-csv/produksi/summary-per-driver" },
            { id: "monitoring-ritase", label: "Monitoring Ritase", icon: "fa-table-columns", href: "/prj-csv/produksi/monitoring-ritase" },
            { id: "transfer-out", label: "Transfer Out", icon: "fa-truck-arrow-right", href: "/prj-csv/produksi/transfer-out" }
          ]
        },
        {
          title: "COLOKAN SOLAR",
          submenus: [
            { id: "pemakaian-solar", label: "Pemakaian Solar", icon: "fa-gas-pump", href: "/prj-csv/colokan-solar/pemakaian-solar" }
          ]
        },
        {
          title: "MASTER DATA",
          submenus: [
            { id: "prior-unit", label: "Prior Unit", icon: "fa-truck-fast", href: "/prj-csv/prior-unit" },
            { id: "closing-finance", label: "Closing Finance", icon: "fa-file-invoice-dollar", href: "/prj-csv/closing-finance" }
          ]
        },
        {
          title: "ADMINISTRATOR",
          submenus: [
            { id: "master-user", label: "Master User", icon: "fa-users-gear", href: "/admin/users" },
            { id: "master-group", label: "Master Group", icon: "fa-users-rectangle", href: "/admin/groups" }
          ]
        }
      ];
    } else {
      // Menu default / PRJ-HO: Hanya Operation Dashboard, SPK Per Cust, dan Warehouse -> Critical Inventory
      menuStructure = [
        {
          title: "OPERATION",
          submenus: [
            { id: "op-dashboard", label: "Operation Dashboard", icon: "fa-gauge-high", href: basePrefix ? basePrefix + "/operation-dashboard" : "/operation-dashboard" }
          ]
        },
        {
          title: "SPK HARIAN",
          submenus: [
            { id: "spk-per-cust", label: "SPK Per Cust", icon: "fa-file-invoice", href: basePrefix ? basePrefix + "/spk-per-cust" : "/spk-per-cust" }
          ]
        },
        {
          title: "WAREHOUSE",
          submenus: [
            { id: "critical-inventory", label: "Critical Inventory", icon: "fa-boxes-stacked", href: basePrefix ? basePrefix + "/warehouse/critical-inventory" : "/prj-ho/warehouse/critical-inventory" }
          ]
        },
        {
          title: "ADMINISTRATOR",
          submenus: [
            { id: "master-user", label: "Master User", icon: "fa-users-gear", href: "/admin/users" },
            { id: "master-group", label: "Master Group", icon: "fa-users-rectangle", href: "/admin/groups" }
          ]
        }
      ];
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
          <a href="/prj-ho" class="flex-1 text-center py-1.5 rounded-md transition ${isHo || (!isCsv && !isHo) ? 'bg-white text-blue-600 font-bold shadow-sm' : 'text-gray-500 hover:text-gray-800'}">
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
        const isActive = sub.label.toLowerCase() === activeMenu.toLowerCase() || sub.id === activeMenu.toLowerCase();
        
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
  }
}

if (!customElements.get('app-sidebar')) {
  customElements.define('app-sidebar', AppSidebar);
}
