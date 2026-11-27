# Backend Base Stack & Architecture Guidelines (`e-com-rental-cloth/backend`)

เอกสารนี้กำหนดโครงสร้างพื้นฐาน (Base Stack) สถาปัตยกรรม **Modular Monolith** และบทบาทของแต่ละ Package ในฝั่ง Go Backend เพื่อให้ทุกฟีเจอร์ในอนาคตเป็นระเบียบ ปลอดภัย รับโหลดสูง และสามารถแยก Instance/Service เฉพาะจุดในอนาคตได้โดยไม่กระทบโค้ดเดิม

## 1. หลักการสำคัญ: เลือกใช้เท่าที่จำเป็น (Modular & Non-Mandatory)
- **ไม่บังคับใช้ครบทุกตัวในทุกฟีเจอร์:** รายการด้านล่างคือเครื่องมือพื้นฐานที่เตรียมไว้ให้ในโปรเจกต์ **ไม่ใช่เช็กลิสต์ที่ทุก Endpoint ต้องเรียกใช้ทั้งหมด** ให้เลือกใช้เฉพาะส่วนที่จำเป็นต่อฟีเจอร์นั้นจริง ๆ (YAGNI)
- **ห้ามลง Library ทับซ้อนกับ Go Standard Library หรือชุดหลัก:**
  - ใช้ `log/slog` ของ Go ทำ JSON Logging ได้เลย (ไม่ต้องลง `zap` หรือ `logrus` เพิ่ม)
  - **ห้ามลง `GORM`** — โปรเจกต์นี้ใช้ `sqlc` + `pgx/v5` เพื่อให้ตรวจสอบความถูกต้องของ SQL และชนิดข้อมูลตั้งแต่ตอน Compile และรองรับฟีเจอร์ช่วงเวลาเช่าของ PostgreSQL ได้เต็มรูปแบบ

---

## 2. บทบาทของแต่ละส่วนใน Backend Base Stack (ใช้ตัวไหนในเรื่องอะไร)

| เครื่องมือ / Package | ใช้สำหรับเรื่องอะไร | เมื่อไหร่ที่ **ไม่จำเป็นต้องใช้** |
| :--- | :--- | :--- |
| **Router & Rate Limit (`go-chi/chi/v5`, `httprate`, `cors`)** | จัดการ HTTP Routing บนมาตรฐาน `net/http` แท้ และตั้ง Rate Limit ผูกกับ `user_id` (ผ่าน `server.RateLimitByUserOrIP`) แยกความเข้มงวดตามความเสี่ยงของ Endpoint | งานประมวลผลภายใน `cmd/worker` หรือการเรียกฟังก์ชันข้าม Module ภายในโปรเซสเดียวกัน |
| **Response & Error Wrapper (`internal/platform/httpx`)** | คืนค่ามาตรฐาน `{ "data": ..., "meta": ... }` และแปลง `AppError` เป็น HTTP Status จริง (`400`, `401`, `403`, `409`, `422`, `429`, `500`) เพื่อให้สอดคล้องกับ `$api` (`onResponseError`) ของฝั่ง Nuxt | ฟังก์ชันระดับ Domain / Service / Repository **ห้ามผูกกับ `http.ResponseWriter`** ให้คืนค่า Go `struct` และ `error` ปกติเท่านั้น |
| **DB & Type-Safe SQL (`jackc/pgx/v5` + `sqlc`)** | เขียนคำสั่ง SQL ในไฟล์ `.sql` แล้วสั่ง `sqlc generate` เพื่อสร้าง Struct และฟังก์ชันเรียก PostgreSQL แบบ Type-safe 100% | เฉพาะหน้าค้นหาชุดเช่าที่มีเงื่อนไข Filter ไม่ตายตัวจำนวนมาก สามารถเสริม SQL Builder บาง ๆ (เช่น `Masterminds/squirrel`) บน `pgxpool` เดียวกันได้ |
| **Migrations (`pressly/goose/v3` + `migrations/`)** | จัดการไฟล์ `.sql` เปลี่ยนแปลงโครงสร้างตารางในโฟลเดอร์ `migrations/` และใช้เป็นต้นทาง Schema ให้ `sqlc` | บน Production **ห้ามรัน `goose down` ย้อนกลับ** เมื่อเกิดปัญหา ให้สร้างไฟล์ Migration ใหม่เดินหน้าอย่างเดียว (Expand & Contract) |
| **Primary Key (`google/uuid` — `UUIDv7`)** | สร้างรหัสหลักของข้อมูลทุกตารางด้วย `id.NewV7()` (`UUIDv7` เรียงตามเวลา) คู่กับคอลัมน์ชนิด `UUID` แท้ใน PostgreSQL (16 bytes) | ตาราง Lookup คงที่ขนาดเล็กมาก (เช่น รหัสสถานะคงที่) ใช้ `TEXT` code อ่านเข้าใจง่ายได้ ไม่จำเป็นต้องใช้ UUID ทุกจุด |
| **Validation & Config (`validator/v10`, `caarlos0/env/v11`)** | ตรวจความถูกต้องของ JSON ขาเข้าผ่าน `validate.DecodeAndValidate` (คืน `422` พร้อมชื่อฟิลด์ตาม JSON Tag) และตรวจค่าบังคับใน `.env` ตั้งแต่วินาทีที่เปิดเครื่อง | ข้อมูลภายในที่ผ่านการสร้างจาก Constructor ของ Domain Struct ที่ตรวจสอบกฎธุรกิจแล้ว ไม่ต้องรัน Struct Tag Validator ซ้ำ |
| **Queue & Async Worker (`riverqueue/river`)** | ส่งงานเบื้องหลัง (ส่งอีเมล/LINE, สร้างใบเสร็จ, คืนคิวใบจองหมดอายุ) เข้าคิวบน PostgreSQL ด้วย `InsertTx` ภายใน Transaction เดียวกับออเดอร์ และรันแยกโปรเซสใน `cmd/worker` | กฎธุรกิจหลักที่ต้องสำเร็จหรือล้มเหลวพร้อมกันทันที (เช่น สร้างใบจองพร้อมตัดโควตาคูปอง) ให้ทำใน Database Transaction เดียวกันตรง ๆ ไม่ต้องโยนเข้า Queue |
| **Auth & Security (`golang.org/x/crypto`, Opaque Session, Turnstile)** | เข้ารหัสรหัสผ่านด้วย `Argon2id`, จัดการ Session แบบส่งผ่าน `HttpOnly` Cookie (เก็บเฉพาะ `SHA-256` ในตาราง `sessions` เพื่อสั่งแบน/เตะบอทได้ทันที), และตรวจตั๋วใช้ครั้งเดียวของ Cloudflare Turnstile ใน `platform/security` | Endpoint อ่านข้อมูลสาธารณะทั่วไป (`GET` หน้าแค็ตตาล็อกชุดเช่า) ไม่ต้องบังคับตรวจ Turnstile หรือ Session |
| **Testing & Security Audit (`testify`, `testcontainers-go`, `govulncheck`, `golangci-lint`)** | เขียน Unit Test เช็กสูตรราคา/วันเช่าด้วย `testify`, เขียน Feature Test ยิง HTTP ทะลุ PostgreSQL จริงด้วย `testutil.NewTestPostgres`, และเช็กช่องโหว่ด้วย `govulncheck` | Unit Test ของ Domain Logic เพียว ๆ ไม่ต้องเสียเวลาบูต Docker Container ของ PostgreSQL ให้รันใน Memory เพื่อความเร็ว |

---

## 3. กฎสถาปัตยกรรม Modular Monolith & ระบบเช่าชุด

1. **กั้นขอบเขต Module เด็ดขาด (`internal/modules/<domain>`):**
   - แบ่งโดเมนตามธุรกิจ เช่น `catalog`, `booking`, `ordering`, `payment`, `notification`
   - ห้าม Module หนึ่งแอบข้ามไปเรียก Repository หรือ Query ตารางของอีก Module ตรง ๆ หากต้องคุยกันให้เรียกผ่าน Public Service Interface เท่านั้น เพื่อให้วันหน้าสามารถแยกบาง Module ออกไปเป็น Instance/Service เดี่ยวได้ทันทีโดยไม่ต้องรื้อโค้ด
   - โฟลเดอร์ `internal/platform/` เป็นเครื่องมือกลาง ห้าม `import` โค้ดจาก `internal/modules/` ย้อนกลับมาเด็ดขาด (บังคับตรวจด้วย `depguard` ใน `.golangci.yml`)

2. **สไตล์ OOP แบบ Idiomatic Go:**
   - ใช้ `struct` คู่กับฟิลด์ตัวพิมพ์เล็ก (Private) + ฟังก์ชันสร้าง `New...()` (Constructor) เพื่อบังคับกฎธุรกิจ (เช่น ราคาเช่า मัดจำ วันเริ่ม-วันคืนต้องสมเหตุสมผลเสมอ)
   - ใช้ `interface` ขนาดเล็กเฉพาะจุดที่ต้องสลับการทำงานหรือกั้นข้าม Module (เช่น `PaymentGateway`) ไม่ต้องสร้าง `IService` มารองรับทุกไฟล์ล่วงหน้า และไม่ต้องใช้ `extends` หรือ DI Framework ซับซ้อน

3. **ป้องกันการจองชุดชนกันระดับ Database Engine (Zero Double-Booking):**
   - ตารางคิวเช่าชุดต้องใช้ความสามารถ `tstzrange` และ `EXCLUDE USING gist` ของ PostgreSQL (เปิด `btree_gist` ไว้แล้วใน `00001_init_extensions_and_sessions.sql`) เพื่อล็อกไม่ให้ชุดรหัสชิ้นเดียวกันมีช่วงวันเช่า + วันซักรีดทับซ้อนกัน แม้มีคำสั่งจองยิงเข้ามาพร้อมกัน

4. **ห้ามเข้ารหัส JSON Response ทับ HTTPS แต่ให้คัดกรองด้วย Response DTO:**
   - ส่งออกเฉพาะฟิลด์ที่หน้าจอต้องแสดงผลจริงเท่านั้น (เช่น ส่งเฉพาะรายการวันที่ไม่ว่าง `blockedDates` ห้ามส่งข้อมูลใบจองของลูกค้าคนอื่นออกไปหน้าบ้าน)
   - ห้ามคืน HTTP `200 OK` เมื่อเกิด Error เด็ดขาด

---

## 4. โครงสร้าง Git Branches, การแยกโฟลเดอร์ Dev / UAT, และการรัน Docker (`APP_ENV`)

### โครงสร้าง Branch & โฟลเดอร์บนเครื่องหลัก (รองรับการทำงานหลายคน มาตรฐานเดียวกับ `frontend/`)
- **Git Branches (สายพานการทำงานจริงสำหรับทีม):**
  กิ่งของนักพัฒนาแต่ละคนหรือกิ่งฟีเจอร์ *(เช่น `<dev-name>`, `dev/<name>`, `feat/*`, `fix/*` — แก้โค้ดบน Local)* -> ดึง `git pull --rebase origin dev` ล่าสุดกันชนกัน -> Merge/PR เข้ากิ่งกลาง `dev` *(รัน CI + Auto-merge เข้า `uat`)* -> `uat` *(กด Deploy เองเพื่อทำ Dev Test / Tester Test)* -> Merge เข้า `main` *(Production)*
- **โฟลเดอร์แยกอิสระ 3 ชุดบนเครื่องหลัก (ป้องกันพอร์ต โค้ด ข้อมูล DB และ Container ชนกัน):**
  - **Local Workspace (`e-com-rental-cloth/backend`):** โฟลเดอร์เขียนโค้ดของนักพัฒนา (เกาะกิ่งทำงาน เช่น `milk` หรือ `feat/*`, ตั้ง `APP_ENV=local`, `PORT=8080`, `POSTGRES_PORT=5432`) เทสบน `localhost:8080` ได้อิสระ
  - **Dev Server Folder (`e-com-rental-cloth-dev/backend`):** ล็อกไว้ที่กิ่ง **`dev`** เท่านั้น (ห้ามเข้าไปแก้โค้ด ตั้ง `APP_ENV=dev`, `PORT=8050`, เชื่อมโดเมน `https://api-dev-rental-cloth.sutiphonglabs.com` ผ่าน Cloudflare Tunnel `rental-cloth-backend-dev-tunnel` และอัปเดตอัตโนมัติทันทีที่ Merge เข้า `dev`)
  - **UAT Server Folder (`e-com-rental-cloth-uat/backend`):** ล็อกไว้ที่กิ่ง **`uat`** เท่านั้น (ห้ามเข้าไปแก้โค้ด ตั้ง `APP_ENV=uat`, `PORT=8180`, เชื่อมโดเมน `https://api-uat-rental-cloth.sutiphonglabs.com` ผ่าน Cloudflare Tunnel `rental-cloth-backend-uat-tunnel` และอัปเดตเฉพาะเมื่อกดปุ่ม `Deploy UAT (Manual)`)

### CI/CD อัตโนมัติบน GitHub Actions (`.github/workflows/`)
- **`ci.yml` (`CI, Auto-Deploy Dev & Sync UAT`):** รองรับทุกกิ่งของนักพัฒนาทุกคนอัตโนมัติ (`branches-ignore: [uat, main]`) โดยรันตรวจ `golangci-lint`, `govulncheck`, `go test`, และคอมไพล์ Binary ทดสอบทุกกิ่งที่ Push และเมื่อรวมโค้ดเข้ากิ่งกลาง `dev` ระบบจะสั่ง **Auto-Deploy ขึ้น Dev Server (`api-dev-rental-cloth.sutiphonglabs.com` พอร์ต `8050`) อัตโนมัติ** พร้อมเช็ก `/api/v1/health` แล้วจึง **Auto-merge จาก `dev` เข้า `uat` มารอไว้โดยยังไม่ Deploy UAT**
- **`deploy-uat.yml` (`Deploy UAT (Manual)`):** ตั้งเป็น `workflow_dispatch` (ไม่รันเอง ต้องกดปุ่ม **Run workflow** ในแท็บ Actions ของ GitHub เท่านั้น) เพื่อสั่ง Self-Hosted Runner บนเครื่องหลักให้ดึงกิ่ง `uat` ล่าสุดไป Build/Restart ชุด Container (`postgres`, `migrate`, `api`, `worker`, `cloudflared`) ขึ้น `https://api-uat-rental-cloth.sutiphonglabs.com` พร้อมตรวจ `/api/v1/health`, อัปเดตสถานะ GitHub Deployments + Job Summary Card, และล้าง Image เก่าอัตโนมัติ

### การสลับโหมด Docker อัตโนมัติด้วย `APP_ENV` (`scripts/docker.go`)
สคริปต์จะตั้งชื่อ Compose Project, Volume ฐานข้อมูล, และ Container แยกตาม `APP_ENV` (`ecom-rental-backend-<env>`) ทำให้รันทั้งชุด Local, Dev Server, และ UAT พร้อมกันบนเครื่องเดียวได้โดยข้อมูลไม่ปนกัน:
- **`APP_ENV=dev`, `APP_ENV=uat`, หรือ `APP_ENV=production`:** ใช้ `compose.yaml` + `compose.prod.yaml` (Multi-stage build คอมไพล์ Static Binary รันบน `gcr.io/distroless/static-debian12:nonroot` ประหยัด RAM เครื่องหลักสูงสุด กินเพียง ~20–30MB ต่อโปรเซส, แยก `postgres` + `migrate` + `api` + `worker`, เปิด `read_only: true`, `cap_drop: ALL`, `no-new-privileges:true`, และไม่เปิดพอร์ต `5432` ออกนอกเครือข่าย Docker)
- **`APP_ENV` เป็นค่าอื่นทั้งหมด (เช่น `local`, `development`):** ใช้ `compose.yaml` + `compose.local.yaml` (รันโหมด Dev พร้อม Hot-reload ด้วย `air`, แยก Volume ของ Go Cache ใน Container ไม่ให้ชนกับ Windows, และเปิดพอร์ต `127.0.0.1:5432` ให้เครื่องพัฒนาต่อตรวจ DB ได้)
- คำสั่งหลัก (รันในโฟลเดอร์ `backend/`):
  - `go run ./scripts/docker.go up --build` (หรือ `up -d --build`)
  - `go run ./scripts/docker.go redeploy` (`git pull` + `up -d --build` สำหรับอัปเดต UAT/Prod)
  - `go run ./scripts/docker.go down`
  - `go run ./scripts/docker.go logs -f`
  - `go run ./scripts/docker.go config`
- ทำงานร่วมกับ Monitoring กลางใน `../infra/compose.monitoring.yaml` (`Uptime Kuma` เช็ก `http://host.docker.internal:8080/api/v1/health` สำหรับ Dev และ `:8180/api/v1/health` สำหรับ UAT พร้อมดู Log รายกล่องใน `Dozzle`)

---

## 5. โครงสร้างโฟลเดอร์อ้างอิงของ `backend/`
- `cmd/api/main.go` — จุดสตาร์ท HTTP API Server (และโหมด `-healthcheck`)
- `cmd/worker/main.go` — จุดสตาร์ท Background Queue Worker (`River`) แยกโปรเซสอิสระจาก API
- `cmd/migrate/main.go` — ตัวรัน Migration ของทั้ง `River` Queue และ `goose` SQL อัตโนมัติก่อนเปิดระบบ
- `internal/platform/` — ตัวกลางพื้นฐาน (`config`, `httpx`, `validate`, `auth`, `security`, `db`, `queue`, `id`)
- `internal/modules/` — โมดูลธุรกิจแต่ละส่วนของระบบเช่าชุด (`catalog`, `booking`, `ordering`, `notification`)
- `internal/server/` — การประกอบ HTTP Router และ Middleware กลาง
- `internal/testutil/` — ตัวช่วยรัน PostgreSQL จริงใน Docker สำหรับทำ Feature Test
- `migrations/` — ไฟล์ SQL Migration (`goose`) และชุดฝังไฟล์ `embed.go`
- `scripts/docker.go` — ตัวสลับไฟล์ Docker Compose อัตโนมัติตาม `APP_ENV` ใน `.env`

---

## 6. แหล่งเก็บเอกสารโปรเจกต์ (Google Drive & Google Docs)
จัดการผ่านปลั๊กอิน `gdrive` (ค้นหา/ดูรายการไฟล์ในโฟลเดอร์) และ `gdocs` (อ่าน/สร้าง/แก้ไขเอกสาร Google Docs โดยระบุ `--folder <FOLDER_ID>` ทุกครั้งที่สร้างเอกสารใหม่):
- **E-Com Rental Cloth (Project Root):** `1sjyITw-_oA7UzhAwHAUMZkBeU5e8_ghw` (https://drive.google.com/drive/folders/1sjyITw-_oA7UzhAwHAUMZkBeU5e8_ghw)
- **SRS (Requirements, API & Feature Specs):** `17ULXLaPIaeNwDS8eD5bZ8tg04bs0PTgL` (https://drive.google.com/drive/folders/17ULXLaPIaeNwDS8eD5bZ8tg04bs0PTgL)
- **Datadic (Data Dictionary & PostgreSQL Schema Docs):** `1Su7ZnF53l-das0KVW8z0z4eO83yZAORH` (https://drive.google.com/drive/folders/1Su7ZnF53l-das0KVW8z0z4eO83yZAORH)
- **Deployment (CI/CD Workflow & System Test URLs):** `1LrwbTggBmG0PbGlwL_Aamslpvs2Bo7YC` (https://drive.google.com/drive/folders/1LrwbTggBmG0PbGlwL_Aamslpvs2Bo7YC)
