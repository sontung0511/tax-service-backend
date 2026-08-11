# Tax Service Backend

Go API cho ứng dụng quản lý thuế hộ kinh doanh và doanh nghiệp. Service dùng `net/http`, tiền là `int64` VND, tax-engine versioned và repository interface. Bản MVP lưu JSON bằng ghi file nguyên tử; có thể thay bằng PostgreSQL mà không sửa handler.

## Chạy local

```bash
make run
```

API chạy tại `http://localhost:8080`. Đăng nhập demo: `demo` / `demo123`.

```bash
curl -s http://localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"demo123"}'
```

Các API ngoài `/healthz` và `/api/login` yêu cầu `Authorization: Bearer mock-session-token`.

## Endpoint

| Method | Path | Chức năng |
|---|---|---|
| POST | `/api/login` | Đăng nhập demo |
| GET/PUT | `/api/profile` | Hồ sơ người nộp thuế |
| GET/POST | `/api/tax-periods` | Danh sách/tạo kỳ |
| POST | `/api/tax-periods/{id}/lock` | Khóa kỳ và lưu snapshot |
| GET/POST | `/api/transactions` | Giao dịch; hỗ trợ `?periodId=` |
| DELETE | `/api/transactions/{id}` | Xóa giao dịch khi kỳ chưa khóa |
| POST | `/api/imports` | Import giao dịch đã preview |
| POST | `/api/calculate` | Tính GTGT/TNCN hộ kinh doanh |
| GET | `/api/exports?periodId=` | Payload phục vụ tạo báo cáo |
| GET | `/api/declarations` | Danh sách tờ khai doanh nghiệp |
| PUT | `/api/declarations/{id}` | Cập nhật/khóa tờ khai doanh nghiệp |
| GET | `/api/audit` | Lịch sử điều chỉnh |

Các tỷ lệ thuế trong `internal/taxengine` là cấu hình prototype, chưa được dùng để nộp hồ sơ thật. XML/mã vạch chưa được sinh cho tới khi có schema chính thức và bộ test đối chiếu HTKK/iTaxViewer.

## Kiểm tra

```bash
make fmt
make vet
make test
```
