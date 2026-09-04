CREATE TABLE IF NOT EXISTS accounts (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    parent_code VARCHAR(20) REFERENCES accounts(code),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_accounts_code_name ON accounts(code, name);

INSERT INTO accounts (code, name, parent_code) VALUES
('111','Tiền mặt',NULL),('1111','- Tiền Việt Nam','111'),('112','Tiền gửi ngân hàng',NULL),('1121','- Tiền Việt Nam','112'),('131','Phải thu của khách hàng',NULL),('133','Thuế GTGT được khấu trừ',NULL),('1331','- Thuế GTGT được khấu trừ của HH-DV','133'),('152','Nguyên liệu, vật liệu',NULL),('154','Chi phí SXKD dở dang',NULL),('156','Hàng hóa',NULL),('1561','- Giá mua hàng hóa','156'),('211','Tài sản cố định',NULL),('2111','- TSCĐ hữu hình','211'),('214','Hao mòn TSCĐ',NULL),('2141','- Hao mòn TSCĐ hữu hình','214'),('242','Chi phí trả trước dài hạn',NULL),('331','Phải trả cho người bán',NULL),('333','Thuế và các khoản phải nộp Nhà nước',NULL),('3331','- Thuế GTGT phải nộp','333'),('3334','- Thuế thu nhập doanh nghiệp','333'),('334','Phải trả người lao động',NULL),('338','Phải trả, phải nộp khác',NULL),('3382','- Kinh phí công đoàn','338'),('3383','- Bảo hiểm xã hội','338'),('411','Nguồn vốn kinh doanh',NULL),('4111','- Vốn đầu tư của chủ sở hữu','411'),('418','Các quỹ thuộc vốn chủ sở hữu',NULL),('421','Lợi nhuận chưa phân phối',NULL),('4211','- Lợi nhuận sau thuế chưa phân phối năm trước','421'),('511','Doanh thu bán hàng và cung cấp dịch vụ',NULL),('5111','- Doanh thu bán Hàng hóa','511'),('5113','- Doanh thu cung cấp Dịch vụ','511'),('515','Doanh thu hoạt động tài chính',NULL),('632','Giá vốn bán hàng',NULL),('6321','- Giá vốn bán hàng Hàng Hóa','632'),('6323','- Giá vốn bán hàng Dịch vụ','632'),('642','Chi phí quản lý kinh doanh',NULL),('6422','- Chi phí quản lý doanh nghiệp','642'),('911','Xác định kết quả kinh doanh',NULL)
ON CONFLICT (code) DO NOTHING;
