# KPI System Backend

REST API untuk KPI System (PT. Cakra Media Data), ditulis dengan Go (Gin + GORM).
Berjalan penuh di lokal dengan MySQL/MariaDB. File upload disimpan di folder `uploads/`.

## Prasyarat
- [Go](https://go.dev/dl/) (versi sesuai `go.mod`)
- MySQL atau MariaDB (contoh: Laragon, XAMPP) yang sedang berjalan di port 3306
- Git

## Instalasi
1. Clone repository dan masuk ke foldernya.
   ```
   git clone <url-repo-backend>
   cd KPI_System_Backend
   ```
2. Buat file `.env` di root folder (contoh ada di `env.local.example`):
   ```env
   DB_DRIVER=mysql
   DB_HOST=127.0.0.1
   DB_PORT=3306
   DB_USER=root
   DB_PASSWORD=
   DB_NAME=kpi_cakra_db

   APP_PORT=8080
   BACKEND_URL=http://localhost:8080
   FRONTEND_URL=http://localhost:3000

   JWT_SECRET=ganti_dengan_secret_anda
   JWT_EXPIRY_HOURS=24
   ENABLE_WHATSAPP=false
   ```
   > Jika `DB_DRIVER` tidak diisi, default-nya `postgres`. Pastikan `DB_DRIVER=mysql`.
3. Unduh dependency:
   ```
   go mod tidy
   ```

## Menjalankan
```
go run main.go
```
Saat pertama kali dijalankan, backend otomatis:
- membuat database (`DB_NAME`) jika belum ada (MySQL),
- membuat/mengupdate semua tabel (AutoMigrate),
- membuat akun admin awal jika belum ada user `superadmin`:
  - username: `superadmin`
  - password: `Admin@1234` (segera ganti setelah login)

API berjalan di `http://localhost:8080` (prefix `/api`).
File upload dapat diakses di `http://localhost:8080/uploads/...`.

## Variabel Environment
| Variabel | Keterangan | Default |
|---|---|---|
| `DB_DRIVER` | `mysql` atau `postgres` | `postgres` |
| `DB_HOST` / `DB_PORT` | Alamat database | `localhost` / `5432` |
| `DB_USER` / `DB_PASSWORD` | Kredensial database | `postgres` / kosong |
| `DB_NAME` | Nama database | `postgres` |
| `DB_SSLMODE` | Khusus Postgres | `disable` |
| `APP_PORT` | Port backend | `8080` |
| `BACKEND_URL` | Dipakai membentuk URL file upload | `http://localhost:8080` |
| `FRONTEND_URL` | Origin frontend | `http://localhost:3000` |
| `JWT_SECRET` | Secret token JWT | (wajib diganti) |
| `ENABLE_WHATSAPP` | Aktifkan integrasi WhatsApp | `false` |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_EMAIL`, `SMTP_PASSWORD` | Email (reset password) | Gmail SMTP |

## Struktur Singkat
- `main.go` - entry point
- `api/` - inisialisasi aplikasi
- `config/` - pembacaan konfigurasi
- `database/` - koneksi, migrasi, seeder
- `controllers/`, `routes/`, `middleware/`, `models/`, `helper/`
- `uploads/` - file hasil upload (tidak di-commit)

## Troubleshooting
- **`bind: Only one usage of each socket address`**: port 8080 sudah dipakai. Hentikan proses lama atau ubah `APP_PORT`.
- **Gagal konek database**: pastikan MySQL berjalan dan nilai `DB_*` di `.env` benar.
- **Login gagal**: gunakan akun yang ada di database, atau `superadmin` jika database baru.
