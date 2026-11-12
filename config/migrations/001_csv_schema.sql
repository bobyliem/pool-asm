-- Skema CSV (imp_sampurna).
-- Dijalankan oleh aplikasi Go. Jangan dijalankan manual.
-- CREATE TABLE memakai IF NOT EXISTS, jadi aman dijalankan ulang.

CREATE TABLE IF NOT EXISTS menu_group (
  id INT NOT NULL AUTO_INCREMENT,
  nama VARCHAR(100) NOT NULL,
  created DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS menu_login (
  id INT NOT NULL AUTO_INCREMENT,
  usrn VARCHAR(100) NOT NULL,
  pass VARCHAR(255) NOT NULL,
  nama VARCHAR(150) NOT NULL,
  aktif TINYINT NOT NULL DEFAULT 1,
  grp_id INT NOT NULL,
  last_login DATETIME NULL,
  createdby VARCHAR(100) NULL,
  created DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_menu_login_usrn (usrn)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS menumaster (
  id INT NOT NULL AUTO_INCREMENT,
  modul VARCHAR(100) NULL,
  nama VARCHAR(150) NULL,
  urut VARCHAR(20) NULL,
  created DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS menu (
  id INT NOT NULL AUTO_INCREMENT,
  mm_id INT NOT NULL,
  nama VARCHAR(150) NULL,
  urut VARCHAR(20) NULL,
  aktif TINYINT NOT NULL DEFAULT 1,
  created DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS menu_access (
  id INT NOT NULL AUTO_INCREMENT,
  grp_id INT NOT NULL,
  menu_id INT NOT NULL,
  created DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS prior_unit (
  id INT NOT NULL AUTO_INCREMENT,
  nopol VARCHAR(50) NOT NULL,
  cabin VARCHAR(50) NULL,
  jenis VARCHAR(100) NULL,
  kategori VARCHAR(100) NULL,
  statue VARCHAR(10) NULL,
  note TEXT NULL,
  created DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ext_closing (
  id INT NOT NULL AUTO_INCREMENT,
  cust VARCHAR(150) NULL,
  cust_code VARCHAR(100) NULL,
  service_order VARCHAR(100) NULL,
  category VARCHAR(100) NULL,
  reff VARCHAR(100) NULL,
  nopol VARCHAR(50) NULL,
  information TEXT NULL,
  released DATETIME NULL,
  completed DATETIME NULL,
  tipe VARCHAR(50) NULL,
  sku VARCHAR(100) NULL,
  prd VARCHAR(255) NULL,
  notes TEXT NULL,
  qty DECIMAL(18,2) NOT NULL DEFAULT 0,
  average DECIMAL(18,2) NOT NULL DEFAULT 0,
  harga DECIMAL(18,2) NOT NULL DEFAULT 0,
  total DECIMAL(18,2) NOT NULL DEFAULT 0,
  statue VARCHAR(50) NULL,
  created DATETIME NULL,
  PRIMARY KEY (id),
  KEY idx_ext_closing_released (released),
  KEY idx_ext_closing_so (service_order)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS pemakaian_solar (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  unit VARCHAR(50) NOT NULL,
  tanggal DATE NOT NULL,
  shift VARCHAR(20) NOT NULL,
  awal_hm DECIMAL(10,2) NOT NULL DEFAULT 0,
  akhir_hm DECIMAL(10,2) NOT NULL DEFAULT 0,
  total_jam_hm DECIMAL(10,2) NOT NULL DEFAULT 0,
  colokan_awal_cm DECIMAL(10,2) NOT NULL DEFAULT 0,
  total_pakai_cm DECIMAL(10,2) NOT NULL DEFAULT 0,
  pakai_liter DECIMAL(10,2) NOT NULL DEFAULT 0,
  colokan_akhir_cm DECIMAL(10,2) NOT NULL DEFAULT 0,
  total_isi_cm DECIMAL(10,2) NOT NULL DEFAULT 0,
  isi_liter DECIMAL(10,2) NOT NULL DEFAULT 0,
  akhir_setelah_isi_cm DECIMAL(10,2) NOT NULL DEFAULT 0,
  sisa_tangki_liter DECIMAL(10,2) NOT NULL DEFAULT 0,
  rasio_liter_per_hm DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at DATETIME NULL,
  updated_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_unit_tgl_shift (unit, tanggal, shift)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS api_stockpile (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  no_tiket VARCHAR(25) NOT NULL DEFAULT '0',
  reff VARCHAR(25) NOT NULL DEFAULT '0',
  nopol VARCHAR(25) NOT NULL DEFAULT '0',
  dk VARCHAR(5) NOT NULL DEFAULT '0',
  prd VARCHAR(25) NOT NULL DEFAULT '0',
  src VARCHAR(25) NOT NULL DEFAULT '0',
  dest VARCHAR(255) NOT NULL DEFAULT '0',
  wsrc DOUBLE NOT NULL DEFAULT 0,
  wdest DOUBLE NOT NULL DEFAULT 0,
  tgl DATETIME NULL,
  updated DATETIME NULL,
  created DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS imp_visitor (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  nopol VARCHAR(50) NULL,
  nobak VARCHAR(50) NULL,
  purpose VARCHAR(100) NULL,
  information TEXT NULL,
  customer VARCHAR(150) NULL,
  cust_code VARCHAR(100) NULL,
  formation VARCHAR(100) NULL,
  planner VARCHAR(100) NULL,
  driver VARCHAR(100) NULL,
  curr_driver VARCHAR(100) NULL,
  tipe VARCHAR(100) NULL,
  category VARCHAR(100) NULL,
  entrydate DATETIME NULL,
  entryby VARCHAR(100) NULL,
  exitdate DATETIME NULL,
  exitby VARCHAR(100) NULL,
  visitor_id VARCHAR(100) NULL,
  sitename VARCHAR(150) NULL,
  created DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS imp_service_order (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  visitor_id VARCHAR(100) NULL,
  so_id VARCHAR(100) NULL,
  service_order VARCHAR(100) NULL,
  category VARCHAR(100) NULL,
  reff VARCHAR(100) NULL,
  customer VARCHAR(150) NULL,
  customer_code VARCHAR(100) NULL,
  site VARCHAR(150) NULL,
  information TEXT NULL,
  plate_number VARCHAR(50) NULL,
  body_number VARCHAR(50) NULL,
  vehicle_type VARCHAR(100) NULL,
  vehicle_category VARCHAR(100) NULL,
  statue VARCHAR(50) NULL,
  released_at DATETIME NULL,
  completed_at DATETIME NULL,
  tipe VARCHAR(50) NULL,
  fulfillment VARCHAR(100) NULL,
  sku VARCHAR(100) NULL,
  product VARCHAR(255) NULL,
  qty DECIMAL(18,2) NOT NULL DEFAULT 0,
  unit_price DECIMAL(18,2) NOT NULL DEFAULT 0,
  total_price DECIMAL(18,2) NOT NULL DEFAULT 0,
  invoice VARCHAR(100) NULL,
  service_order_id VARCHAR(100) NULL,
  last_notes TEXT NULL,
  created DATETIME NULL,
  PRIMARY KEY (id),
  KEY idx_iso_site (site),
  KEY idx_iso_customer (customer_code),
  KEY idx_iso_plate (plate_number),
  KEY idx_iso_released (released_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ss_pok (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tgl_sj DATETIME NULL,
  nopol VARCHAR(50) NULL,
  sopir VARCHAR(100) NULL,
  qty_src DECIMAL(18,2) NOT NULL DEFAULT 0,
  qty_dest DECIMAL(18,2) NOT NULL DEFAULT 0,
  statue VARCHAR(50) NULL,
  PRIMARY KEY (id),
  KEY idx_ss_pok_tgl (tgl_sj),
  KEY idx_ss_pok_nopol (nopol)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS log_imp (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  started DATETIME NULL,
  finished DATETIME NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

INSERT INTO menu_group (id, nama, created)
SELECT 1, 'SUPERADMIN', NOW()
WHERE NOT EXISTS (SELECT 1 FROM menu_group WHERE id = 1);
