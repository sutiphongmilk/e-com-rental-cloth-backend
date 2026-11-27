# E-Com Rental Cloth — Backend (`Go Modular Monolith`)

ระบบหลังบ้านของแพลตฟอร์มเช่าเสื้อผ้า พัฒนาด้วย **Go 1.27**, **Chi Router**, **PostgreSQL 17 (`pgx/v5` + `sqlc` + `goose`)**, และ **River Queue** ในรูปแบบ **Modular Monolith** ที่แยกรัน `api` และ `worker` ออกจากกันได้อย่างอิสระ

ดูกฎสถาปัตยกรรมและหน้าที่ของแต่ละ Package ได้ใน [`GEMINI.md`](./GEMINI.md)

## โครงสร้างหลัก
- `cmd/api` — HTTP API Server (รองรับ `-healthcheck`)
- `cmd/worker` — Background Queue Worker (`River`) แยกโปรเซสจาก API
- `cmd/migrate` — รัน Migration ของทั้ง `River` Queue และ `goose` SQL อัตโนมัติ
- `internal/platform` — ตัวกลางพื้นฐาน (`config`, `httpx`, `validate`, `auth`, `security`, `db`, `queue`, `id`)
- `internal/modules` — โมดูลธุรกิจแต่ละส่วนของระบบเช่าชุด
- `migrations` — ไฟล์ SQL Migration (`goose`)
- `scripts/docker.go` — ตัวรัน Docker Compose สลับโหมด `local` / `production` อัตโนมัติตามค่า `APP_ENV` ใน `.env`

## โครงสร้าง Git Branches, CI/CD & การแยกโฟลเดอร์ 3 ชุดบนเครื่องหลัก
- **สายพาน Git Branches:** กิ่งนักพัฒนา/ฟีเจอร์ (`milk`, `<dev-name>`, `feat/*`, `fix/*`) -> `git pull --rebase origin dev` -> Merge/PR เข้า `dev` *(รัน CI -> **Auto-Deploy ขึ้น Dev Server** -> Auto-merge เข้า `uat` โดยยังไม่ Deploy UAT)* -> กดรัน **`Deploy UAT (Manual)`** บน GitHub Actions เพื่ออัปเดต UAT -> Merge เข้า `main` *(Production)*
- **Local Workspace:** `e-com-rental-cloth/backend` (เกาะกิ่ง `milk`, `APP_ENV=local`, `PORT=8080`, `POSTGRES_PORT=5432`)
- **Dev Server Folder:** `e-com-rental-cloth-dev/backend` (ล็อกกิ่ง `dev`, `APP_ENV=dev`, `PORT=8050`, Public URL: `https://api-dev-rental-cloth.sutiphonglabs.com` ผ่าน Cloudflare Tunnel)
- **UAT Server Folder:** `e-com-rental-cloth-uat/backend` (ล็อกกิ่ง `uat`, `APP_ENV=uat`, `PORT=8180`, Public URL: `https://api-uat-rental-cloth.sutiphonglabs.com` ผ่าน Cloudflare Tunnel)
- **GitHub Actions (`.github/workflows/`):**
  - `ci.yml` — รัน `golangci-lint`, `govulncheck`, `go test`, และคอมไพล์ Binary ทดสอบทุกกิ่งที่ Push พร้อม **Auto-Deploy ขึ้น Dev Server (`:8050`)** และ Auto-merge `dev -> uat` เมื่อรวมเข้า `dev`
  - `deploy-uat.yml` — กดรันแบบ Manual (`workflow_dispatch`) ผ่าน Self-Hosted Runner บนเครื่องหลักเพื่ออัปเดต UAT Container และตรวจ `/api/v1/health`

## การรันผ่าน Docker
คัดลอก `.env.example` เป็น `.env` แล้วรันคำสั่ง:
```powershell
go run ./scripts/docker.go up -d --build
go run ./scripts/docker.go redeploy
go run ./scripts/docker.go logs -f
go run ./scripts/docker.go down
```
- เมื่อ `APP_ENV=dev`, `APP_ENV=uat`, หรือ `APP_ENV=production` จะใช้ `compose.yaml` + `compose.prod.yaml` (`Distroless` static binary, `read_only: true`, `cap_drop: ALL`, แยก Compose Project/DB Volume ไม่ชนกัน)
- เมื่อ `APP_ENV` เป็นค่าอื่น (เช่น `local`) จะใช้ `compose.yaml` + `compose.local.yaml` (Hot-reload ด้วย `air` และเปิดพอร์ต `127.0.0.1:5432`)

## คำสั่งพัฒนาและตรวจสอบคุณภาพโค้ด
```powershell
sqlc generate
go test -v ./...
golangci-lint run ./...
govulncheck ./...
```
