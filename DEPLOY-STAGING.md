# Deploy ASM ke server staging (CI/CD)

Satu alur: push ke branch `staging` di GitHub → GitHub Actions build image → image masuk GitHub Container Registry → server staging menarik image dan menjalankan `docker compose`.

Aplikasi yang naik: container `asm-app` (port 8011) dan MySQL `imp_sampurna` (hanya di jaringan Docker, tidak dibuka ke internet). Database Head Office tetap remote lewat `DB_HO_*` di `.env` server.

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
                     /var/www/pool-asm
```

Lakukan bagian 1–6 sekali. Setelah itu setiap push ke `staging` deploy sendiri (bagian 7).

---

## 1. Bereskan repo sebelum pipeline pertama

Pipeline gagal kalau langkah ini dilewati.

1. Ubah baris pertama `Dockerfile` dari `golang:1.24-alpine` menjadi `golang:1.26-alpine`. `go.mod` memakai Go 1.26. Image 1.24 tidak bisa meng-compile modul itu.
2. Jangan memakai `.env` yang sudah ada di GitHub. File itu ikut ter-commit, jadi password di dalamnya anggap sudah bocor. Password MySQL staging yang baru ada di `.staging-secrets` (tidak di-commit). Salin ke `.env` server pada bagian 4. Password user HO di `10.2.1.2` belum diganti: dari mesin ini tidak ada akses admin ke MySQL itu, dan menggantinya akan memutus aplikasi yang masih memakai user lama.
3. Skema tabel ada di `config/migrations/001_csv_schema.sql`. Aplikasi menjalankannya sendiri lewat `go run . migrate` atau saat server nyala. Bukan dump data produksi.
4. Buat branch `staging` dari commit yang sudah berisi perbaikan Dockerfile, lalu push:

```bash
git checkout -b staging
git push -u origin staging
```

---

## 2. Siapkan server staging

Server yang dipakai adalah server yang sudah ada. Semua proyek di server itu tinggal di `/var/www`, jadi ASM diletakkan di `/var/www/pool-asm`. Jangan buat folder baru di `/opt`.

Asumsi: Ubuntu 22.04 atau 24.04, akses SSH, IP yang bisa dijangkau GitHub Actions. Lewati pemasangan Docker kalau `docker compose version` sudah berhasil.

Masuk sebagai user dengan sudo:

```bash
ssh -p <SSH_PORT> <user>@<IP_STAGING>
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

Pakai user `deploy` yang sudah ada. Buat user itu hanya kalau belum ada, lalu izinkan menjalankan Docker dan milikkan folder proyek:

```bash
id deploy >/dev/null 2>&1 || sudo adduser --disabled-password --gecos "" deploy
sudo usermod -aG docker deploy
sudo mkdir -p /var/www/pool-asm
sudo chown deploy:deploy /var/www/pool-asm
```

Buka hanya SSH dan aplikasi. MySQL tidak dibuka ke luar:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 8011/tcp
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

`/home/deploy/.ssh/authorized_keys` sudah ada dan dipakai CI/CD GitLab proyek lain. Jangan dihapus atau ditimpa. Tambahkan satu baris kunci GitHub Actions di bawah kunci yang sudah ada.

User `deploy` tidak punya password, jadi `ssh-copy-id deploy@...` akan selalu menanyakan password. Pasang lewat root, dari laptop, di folder yang berisi `asm-staging-deploy.pub`:

`<SSH_PORT>` adalah port SSH server itu, bukan 22.

```bash
ssh -p <SSH_PORT> root@<IP_STAGING> 'mkdir -p /home/deploy/.ssh && touch /home/deploy/.ssh/authorized_keys && chmod 700 /home/deploy/.ssh && chmod 600 /home/deploy/.ssh/authorized_keys'
grep -qxF "$(cat ./asm-staging-deploy.pub)" <(ssh -p <SSH_PORT> root@<IP_STAGING> 'cat /home/deploy/.ssh/authorized_keys') || cat ./asm-staging-deploy.pub | ssh -p <SSH_PORT> root@<IP_STAGING> 'cat >> /home/deploy/.ssh/authorized_keys'
ssh -p <SSH_PORT> root@<IP_STAGING> 'chown -R deploy:deploy /home/deploy/.ssh'
```

Perintah kedua hanya menambahkan baris kalau kunci itu belum ada. Kunci GitLab yang sudah tersimpan tetap di tempatnya.

Tes, harus masuk tanpa password:

```bash
ssh -p <SSH_PORT> -i ./asm-staging-deploy deploy@<IP_STAGING>
```

Isi private key (`asm-staging-deploy`, bukan file `.pub`) nanti disimpan sebagai secret `STAGING_SSH_KEY`. Jangan di-commit.

---

## 4. File yang hidup di server

`docker-compose.staging.yml` ada di repo. Server mengambil file itu lewat `git clone` atau `git pull`. Yang ditulis manual di server hanya `.env`, karena berisi password.

### 4.1 `/var/www/pool-asm/.env`

Ganti setiap nilai. `DB_CSV_HOST` harus `db-csv` karena itu nama service di Compose, bukan `localhost`.

```ini
PORT=8011
GIN_MODE=release
TZ=Asia/Jakarta

DB_CSV_HOST=db-csv
DB_CSV_PORT=3306
DB_CSV_USER=asm
DB_CSV_PASS=GANTI_PASSWORD_USER_APLIKASI
DB_CSV_NAME=imp_sampurna
DB_ROOT_PASS=PASSWORD_ROOT_YANG_SUDAH_ADA

# Akun ini sudah ada di server Head Office. Migrasi tidak membuatnya.
# Kosongkan kalau kredensial HO belum ada. Dashboard CSV tetap jalan.
DB_HO_HOST=10.2.1.2
DB_HO_PORT=3306
DB_HO_USER=
DB_HO_PASS=
DB_HO_NAME=workshop

ASM_IMAGE=ghcr.io/bobyliem/pool-asm:staging

# User login awal. Hapus dua baris ini setelah bisa login.
ASM_BOOTSTRAP_USER=staging
ASM_BOOTSTRAP_PASS=GANTI_PASSWORD_ADMIN
```

Aplikasi konek sebagai `asm`, bukan `root`. `DB_ROOT_PASS` adalah password root yang sudah ada di volume MySQL. Jangan diubah. `DB_CSV_PASS` adalah password user `asm`. Di server yang sudah jalan, isi `DB_ROOT_PASS` dengan password root yang sekarang, lalu ganti `DB_CSV_USER` menjadi `asm`.

`DB_HO_USER` dan `DB_HO_PASS` dikosongkan dulu. User itu ada di MySQL Head Office (`10.2.1.2`), bukan di database staging, dan migrasi tidak membuatnya. Isi kedua baris itu hanya kalau kamu sudah punya username dan password yang memang sudah dibuat di server HO.

### 4.2 `docker-compose.staging.yml`

File ini ada di root repo dan ikut di-commit. Jangan salin isinya manual ke server.

Password di healthcheck dibaca dari environment Compose. `MYSQL_ROOT_PASSWORD` mengambil `DB_CSV_PASS` dari `.env` yang satu folder dengan file Compose. Port aplikasi di host adalah `8011`. Port MySQL tidak dipetakan ke host. Database hanya bisa diakses container `asm-app`.

Clone pertama, dari server:

```bash
sudo mkdir -p /var/www/pool-asm
sudo chown deploy:deploy /var/www/pool-asm
git clone -b staging https://github.com/bobyliem/pool-asm.git /var/www/pool-asm
```

Kalau folder itu sudah ada dan masih kosong, jalankan `git clone` yang sama. `.env` dibuat setelah clone, di dalam folder itu.

---

## 5. Token GitHub dan secrets

Image disimpan di GitHub Container Registry paket `bobyliem/pool-asm`. Server butuh token baca paket itu.

Jangan memakai fine-grained token. Resource owner-nya hanya akun yang sedang login (`taufikibo`), dan daftar repository di sana hanya repo milik akun itu. `bobyliem/pool-asm` tidak akan muncul, dan permission `Packages` untuk repo itu juga tidak ada.

Buat token classic:

1. GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic) → Generate new token (classic).
2. Note: `pool-asm staging pull`.
3. Centang scope `read:packages` saja. Jangan centang `repo` kalau tidak perlu.
4. Generate token, salin sekali.

Token ini milik `taufikibo`. Username login registry juga `taufikibo`, bukan `bobyliem`.

Di repo GitHub, halaman `bobyliem/pool-asm`: **Settings → Secrets and variables → Actions → New repository secret**. Buat persis nama ini:

| Secret | Isi |
|---|---|
| `STAGING_HOST` | IP atau hostname server |
| `STAGING_PORT` | Port SSH server. Bukan `22` kalau server memakai port lain |
| `STAGING_USER` | `deploy` |
| `STAGING_SSH_KEY` | isi penuh file private key `asm-staging-deploy`, termasuk baris `BEGIN` dan `END` |
| `GHCR_USER` | `taufikibo` |
| `GHCR_READ_TOKEN` | token classic di atas |

`GITHUB_TOKEN` tidak perlu dibuat manual. Actions sudah menyediakannya untuk push image. Yang login ke registry dari server adalah `GHCR_USER` dan `GHCR_READ_TOKEN`, bukan user `deploy`.

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
          port: ${{ secrets.STAGING_PORT }}
          username: ${{ secrets.STAGING_USER }}
          key: ${{ secrets.STAGING_SSH_KEY }}
          script_stop: true
          script: |
            set -euo pipefail
            cd /var/www/pool-asm
            git pull --ff-only origin staging
            echo "${{ secrets.GHCR_READ_TOKEN }}" | docker login ghcr.io -u ${{ secrets.GHCR_USER }} --password-stdin
            docker compose --env-file .env -f docker-compose.staging.yml pull app
            docker compose --env-file .env -f docker-compose.staging.yml up -d
            docker logout ghcr.io
            docker image prune -f
```

Push file workflow ke `staging`. Job pertama akan build lalu SSH. Saat container aplikasi nyala, program Go membuat database `imp_sampurna`, user MySQL `asm`, dan tabelnya dari `config/migrations/001_csv_schema.sql`. Tidak perlu menjalankan file SQL.

### Database

Di laptop, dari folder repo, dengan `.env` yang sudah terisi:

```bash
go run . migrate
```

Di server, perintah yang sama lewat container. Jalankan setelah `docker compose up` kalau ingin mengulang migrasi tanpa menyalakan ulang proses:

```bash
cd /var/www/pool-asm
docker compose --env-file .env -f docker-compose.staging.yml exec app /app/asm migrate
```

Perintah itu hanya membuat database dan tabel yang belum ada. Data yang sudah tersimpan tidak dihapus.

User login awal dibuat kalau `.env` berisi `ASM_BOOTSTRAP_USER` dan `ASM_BOOTSTRAP_PASS`. Kosongkan keduanya setelah user berhasil login. Group `SUPERADMIN` ikut dibuat oleh migrasi.

### Cek hasil

Di server:

```bash
cd /var/www/pool-asm
docker compose --env-file .env -f docker-compose.staging.yml ps
docker compose --env-file .env -f docker-compose.staging.yml logs --tail=50 app
```

Dari laptop:

```bash
curl -I http://<IP_STAGING>:8011/
```

Harapan: HTTP 307 ke `/prj-ho/operation-dashboard`. Browser: `http://<IP_STAGING>:8011/login`.

Log aplikasi yang sehat memuat baris koneksi ke MySQL CSV. Koneksi HO gagal hanya kalau jaringan ke `10.2.1.2` belum dibuka. Itu tidak menggagalkan container.

---

## 7. Deploy berikutnya

1. Kerjakan di branch lain, merge atau cherry-pick ke `staging`.
2. `git push origin staging`.
3. Buka tab **Actions** di GitHub. Job `Build image` lalu `Deploy ke server` harus hijau.
4. Buka lagi `http://<IP_STAGING>:8011/login`.

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
cd /var/www/pool-asm
# di .env, sementara:
# ASM_IMAGE=ghcr.io/bobyliem/pool-asm:sha-<40-karakter-commit>
docker login ghcr.io -u taufikibo
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
| `denied` saat server `docker pull` | Token tidak punya `read:packages`, atau username login bukan pemilik token | Bagian 5. Username login adalah `GHCR_USER` (`taufikibo`), bukan `bobyliem` |
| SSH `Permission denied` | Private key secret tidak lengkap, atau public key belum di `deploy` | Ulangi bagian 3. Secret harus isi file tanpa `.pub` |
| `asm-app` restart loop | `.env` salah, atau MySQL belum healthy | `docker compose ... logs app` dan `logs db-csv` |
| Halaman CSV kosong / query error | Migrasi belum jalan, atau MySQL belum healthy | `docker compose ... logs app`, lalu `docker compose ... exec app /app/asm migrate` |
| Endpoint HO 500 "belum terhubung" | Server tidak sampai ke `10.2.1.2:3306` | Buka firewall kantor untuk IP staging, atau isi `DB_HO_*` yang bisa dijangkau |
| Sync CSV dari UI gagal | Fitur itu memanggil `php.exe` path Windows | Biarkan di luar staging. Staging ini menjalankan binary Go di Linux |

Perintah yang aman di server saat memperbaiki:

```bash
cd /var/www/pool-asm
docker compose --env-file .env -f docker-compose.staging.yml logs -f app
```

Jangan menjalankan `docker compose down -v`. Flag `-v` menghapus volume MySQL.
