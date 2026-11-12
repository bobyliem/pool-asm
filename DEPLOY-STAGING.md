# Deploy ASM ke server staging (CI/CD)

Satu alur: push ke branch `staging` di GitHub → GitHub Actions build image → image masuk GitHub Container Registry → server staging menarik image dan menjalankan `docker compose`.

Aplikasi yang naik: container `asm-app` (port 8080) dan MySQL `imp_sampurna` (hanya di jaringan Docker, tidak dibuka ke internet). Database Head Office tetap remote lewat `DB_HO_*` di `.env` server.

```
laptop  --push staging-->  GitHub
                              |
                              v
                        GitHub Actions
                        build + push image
                              |
                              v
                        ghcr.io/...:staging
                              |
                         SSH deploy
                              |
                              v
                     server staging
                     /opt/asm-staging
```

Lakukan bagian 1–6 sekali. Setelah itu setiap push ke `staging` deploy sendiri (bagian 7).

---

## 1. Bereskan repo sebelum pipeline pertama

Pipeline gagal kalau langkah ini dilewati.

1. Ubah baris pertama `Dockerfile` dari `golang:1.24-alpine` menjadi `golang:1.26-alpine`. `go.mod` memakai Go 1.26. Image 1.24 tidak bisa meng-compile modul itu.
2. Jangan memakai `.env` yang sudah ada di GitHub. File itu ikut ter-commit, jadi password di dalamnya anggap sudah bocor. Password MySQL staging yang baru ada di `.staging-secrets` (tidak di-commit). Salin ke `.env` server pada bagian 4. Password user HO di `10.2.1.2` belum diganti: dari mesin ini tidak ada akses admin ke MySQL itu, dan menggantinya akan memutus aplikasi yang masih memakai user lama.
3. File skema ada di `imp_sampurna.sql` di folder repo ini (diabaikan Git lewat `*.sql`). Isinya tabel kosong yang disusun dari model aplikasi, bukan dump data produksi, karena dump asli tidak ada di repo maupun di mesin ini. Import dilakukan sekali di bagian 6.
4. Buat branch `staging` dari commit yang sudah berisi perbaikan Dockerfile, lalu push:

```bash
git checkout -b staging
git push -u origin staging
```

---

## 2. Siapkan server staging

Asumsi: Ubuntu 22.04 atau 24.04, akses SSH, IP publik atau IP yang bisa dijangkau GitHub Actions.

Masuk sebagai user dengan sudo:

```bash
ssh <user>@<IP_STAGING>
```

Pasang Docker Engine dan plugin Compose:

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

Buat user khusus deploy (bukan root) dan izinkan menjalankan Docker:

```bash
sudo adduser --disabled-password --gecos "" deploy
sudo usermod -aG docker deploy
sudo mkdir -p /opt/asm-staging
sudo chown deploy:deploy /opt/asm-staging
```

Buka hanya SSH dan aplikasi. MySQL tidak dibuka ke luar:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 8080/tcp
sudo ufw enable
```

Cek:

```bash
docker --version
docker compose version
```

---

## 3. Buat kunci SSH untuk GitHub Actions

Di laptop (bukan di server), buat kunci yang hanya dipakai pipeline:

```bash
ssh-keygen -t ed25519 -f ./asm-staging-deploy -C "github-actions-asm-staging" -N ""
```

Pasang public key ke user `deploy` di server:

```bash
ssh-copy-id -i ./asm-staging-deploy.pub deploy@<IP_STAGING>
```

Tes, harus masuk tanpa password:

```bash
ssh -i ./asm-staging-deploy deploy@<IP_STAGING>
```

Isi private key (`asm-staging-deploy`, bukan file `.pub`) nanti disimpan sebagai secret `STAGING_SSH_KEY`. Jangan di-commit.

---

## 4. File yang hidup di server

Login sebagai `deploy`, lalu tulis dua file ini. Keduanya tidak masuk Git.

### 4.1 `/opt/asm-staging/.env`

Ganti setiap nilai. `DB_CSV_HOST` harus `db-csv` karena itu nama service di Compose, bukan `localhost`.

```ini
PORT=8080
GIN_MODE=release
TZ=Asia/Jakarta

DB_CSV_HOST=db-csv
DB_CSV_PORT=3306
DB_CSV_USER=root
DB_CSV_PASS=GANTI_PASSWORD_MYSQL_STAGING
DB_CSV_NAME=imp_sampurna

DB_HO_HOST=10.2.1.2
DB_HO_PORT=3306
DB_HO_USER=GANTI_USER_HO
DB_HO_PASS=GANTI_PASSWORD_HO
DB_HO_NAME=workshop

ASM_IMAGE=ghcr.io/bobyliem/pool-asm:staging
```

`DB_HO_*` hanya terisi kalau server staging bisa mencapai `10.2.1.2:3306`. Kalau tidak, aplikasi tetap nyala, endpoint HO mengembalikan error JSON, dashboard CSV tetap jalan setelah database lokal di-import.

### 4.2 `/opt/asm-staging/docker-compose.staging.yml`

Password di healthcheck dibaca dari environment Compose, bukan ditulis ulang di file ini. Baris `MYSQL_ROOT_PASSWORD` mengambil `DB_CSV_PASS` dari `.env` yang satu folder dengan file Compose.

```yaml
services:
  db-csv:
    image: mysql:8.0
    container_name: asm-mysql-csv
    restart: unless-stopped
    command: --default-authentication-plugin=mysql_native_password --sql_mode=STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_CSV_PASS}
      MYSQL_DATABASE: ${DB_CSV_NAME}
      TZ: Asia/Jakarta
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD-SHELL", "mysqladmin ping -h localhost -uroot -p$$MYSQL_ROOT_PASSWORD --silent"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 40s
    networks:
      - asm-net

  app:
    image: ${ASM_IMAGE}
    container_name: asm-app
    restart: unless-stopped
    depends_on:
      db-csv:
        condition: service_healthy
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      DB_CSV_HOST: db-csv
      DB_CSV_PORT: "3306"
      TZ: Asia/Jakarta
    networks:
      - asm-net

volumes:
  mysql_data:

networks:
  asm-net:
    driver: bridge
```

Port `3307` sengaja tidak dipetakan. Database hanya bisa diakses container `asm-app`.

---

## 5. Token GitHub dan secrets

Image disimpan di GitHub Container Registry paket `bobyliem/pool-asm`. Server butuh token baca paket itu.

1. GitHub → Settings → Developer settings → Personal access tokens → Fine-grained token.
2. Resource owner: `bobyliem`. Repository access: hanya `pool-asm`.
3. Permission: `Packages` = Read-only.
4. Salin token sekali. Namai di langkah secret sebagai `GHCR_READ_TOKEN`.

Di repo GitHub: **Settings → Secrets and variables → Actions → New repository secret**. Buat persis nama ini:

| Secret | Isi |
|---|---|
| `STAGING_HOST` | IP atau hostname server |
| `STAGING_USER` | `deploy` |
| `STAGING_SSH_KEY` | isi penuh file private key `asm-staging-deploy`, termasuk baris `BEGIN` dan `END` |
| `GHCR_READ_TOKEN` | token fine-grained di atas |

`GITHUB_TOKEN` tidak perlu dibuat manual. Actions sudah menyediakannya untuk push image.

Kalau organisasi memakai GitHub Free dan paket GHCR private, pastikan user `deploy` tidak perlu akun GitHub: yang login ke registry adalah token di atas, saat job deploy berjalan.

---

## 6. Workflow CI/CD

Commit file ini di branch `staging`: `.github/workflows/staging.yml`

```yaml
name: Deploy staging

on:
  push:
    branches: [staging]
  workflow_dispatch:

permissions:
  contents: read
  packages: write

jobs:
  build:
    name: Build image
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: docker/setup-buildx-action@v3

      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ghcr.io/bobyliem/pool-asm:staging
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    name: Deploy ke server
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Pull image dan nyalakan compose
        uses: appleboy/ssh-action@v1.2.0
        with:
          host: ${{ secrets.STAGING_HOST }}
          username: ${{ secrets.STAGING_USER }}
          key: ${{ secrets.STAGING_SSH_KEY }}
          script_stop: true
          script: |
            set -euo pipefail
            cd /opt/asm-staging
            echo "${{ secrets.GHCR_READ_TOKEN }}" | docker login ghcr.io -u bobyliem --password-stdin
            docker compose --env-file .env -f docker-compose.staging.yml pull app
            docker compose --env-file .env -f docker-compose.staging.yml up -d
            docker logout ghcr.io
            docker image prune -f
```

Push file workflow ke `staging`. Job pertama akan build lalu SSH. MySQL nyala kosong sampai import di bawah selesai.

### Import database sekali

Dari laptop, setelah container `asm-mysql-csv` statusnya healthy:

```bash
scp -i ./asm-staging-deploy imp_sampurna.sql deploy@<IP_STAGING>:/tmp/imp_sampurna.sql
ssh -i ./asm-staging-deploy deploy@<IP_STAGING>
cd /opt/asm-staging
docker compose --env-file .env -f docker-compose.staging.yml exec -T db-csv \
  sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" imp_sampurna' < /tmp/imp_sampurna.sql
rm /tmp/imp_sampurna.sql
```

Ulangi import hanya kalau volume `mysql_data` dihapus. Deploy berikutnya tidak mengosongkan data.

### Cek hasil

Di server:

```bash
cd /opt/asm-staging
docker compose --env-file .env -f docker-compose.staging.yml ps
docker compose --env-file .env -f docker-compose.staging.yml logs --tail=50 app
```

Dari laptop:

```bash
curl -I http://<IP_STAGING>:8080/
```

Harapan: HTTP 307 ke `/prj-ho/operation-dashboard`. Browser: `http://<IP_STAGING>:8080/login`.

Log aplikasi yang sehat memuat baris koneksi ke MySQL CSV. Koneksi HO gagal hanya kalau jaringan ke `10.2.1.2` belum dibuka. Itu tidak menggagalkan container.

---

## 7. Deploy berikutnya

1. Kerjakan di branch lain, merge atau cherry-pick ke `staging`.
2. `git push origin staging`.
3. Buka tab **Actions** di GitHub. Job `Build image` lalu `Deploy ke server` harus hijau.
4. Buka lagi `http://<IP_STAGING>:8080/login`.

Deploy manual tanpa push: tab Actions → Deploy staging → Run workflow.

---

## 8. Rollback

GHCR menyimpan image lewat tag `staging`, dan tag itu tertimpa tiap deploy. Untuk bisa mundur, tambahkan satu tag SHA di langkah build (ganti blok `tags` saja):

```yaml
tags: |
  ghcr.io/bobyliem/pool-asm:staging
  ghcr.io/bobyliem/pool-asm:sha-${{ github.sha }}
```

Rollback di server, ganti SHA dengan commit yang sebelumnya sehat:

```bash
cd /opt/asm-staging
# di .env, sementara:
# ASM_IMAGE=ghcr.io/bobyliem/pool-asm:sha-<40-karakter-commit>
docker login ghcr.io -u bobyliem
docker compose --env-file .env -f docker-compose.staging.yml pull app
docker compose --env-file .env -f docker-compose.staging.yml up -d
docker logout ghcr.io
```

Kembalikan `ASM_IMAGE` ke `ghcr.io/bobyliem/pool-asm:staging` setelah branch `staging` di-push ulang ke commit yang benar. Kalau tidak, deploy berikutnya menimpa lagi dengan image lama.

Data MySQL tidak ikut ter-rollback. Volume `mysql_data` tetap.

---

## 9. Kalau job gagal

| Gejala | Penyebab | Perbaikan |
|---|---|---|
| Build gagal di `go mod download` / compile | Dockerfile masih Go 1.24 | Bagian 1, lalu push ulang |
| `denied` saat push GHCR | Workflow tanpa `packages: write` | Pakai file YAML di bagian 6 utuh |
| `denied` saat server `docker pull` | Token tidak punya Packages Read, atau login user salah | Bagian 5. Username login di script adalah `bobyliem` |
| SSH `Permission denied` | Private key secret tidak lengkap, atau public key belum di `deploy` | Ulangi bagian 3. Secret harus isi file tanpa `.pub` |
| `asm-app` restart loop | `.env` salah, atau MySQL belum healthy | `docker compose ... logs app` dan `logs db-csv` |
| Halaman CSV kosong / query error | Dump belum di-import | Bagian 6, import SQL |
| Endpoint HO 500 "belum terhubung" | Server tidak sampai ke `10.2.1.2:3306` | Buka firewall kantor untuk IP staging, atau isi `DB_HO_*` yang bisa dijangkau |
| Sync CSV dari UI gagal | Fitur itu memanggil `php.exe` path Windows | Biarkan di luar staging. Staging ini menjalankan binary Go di Linux |

Perintah yang aman di server saat memperbaiki:

```bash
cd /opt/asm-staging
docker compose --env-file .env -f docker-compose.staging.yml logs -f app
```

Jangan menjalankan `docker compose down -v`. Flag `-v` menghapus volume MySQL.
