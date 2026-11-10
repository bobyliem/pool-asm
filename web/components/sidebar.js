// Modul Component Sidebar Navigation (<app-sidebar></app-sidebar>)
class AppSidebar extends HTMLElement {
  connectedCallback() {
    this.className = 'flex flex-col h-full max-h-full overflow-hidden shrink-0';
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
            { id: "summary-per-cust", label: "Summary Per Cust", icon: "fa-table-cells-large", href: "/prj-csv/summary-per-cust" },
            { id: "resume-per-kategori", label: "Resume Per Kategori", icon: "fa-border-all", href: "/prj-csv/resume-per-kategori" }
          ]
        },
        {
          title: "PRODUKSI",
          submenus: [
            { id: "resume-per-hari", label: "Resume Per Hari", icon: "fa-calendar-day", href: "/prj-csv/produksi/resume-per-hari" },
            { id: "summary-per-driver", label: "Summary Per Driver", icon: "fa-user-tie", href: "/prj-csv/produksi/summary-per-driver" },
            { id: "monitoring-ritase", label: "Monitoring Ritase", icon: "fa-table-columns", href: "/prj-csv/produksi/monitoring-ritase" }
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
      <aside id="sidebarNav" class="w-64 bg-white border-r border-gray-200 p-4 pb-12 flex flex-col space-y-4 overflow-y-auto h-full max-h-full shrink-0 transition-all duration-300" style="scrollbar-width: thin; scrollbar-color: #cbd5e1 #f8fafc;">
        
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
               data-menu-id="${sub.id}"
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
