-- Contract Counter (Stateful)

-- Hàm này được gọi khi Deploy
function initialize()
    db_put("counter", 0)
end

-- Tăng giá trị counter
function increment()
    local current_val = db_get("counter")
    local new_val = tonumber(current_val) + 1
    db_put("counter", new_val)
end

-- Chạy hàm initialize khi file được load (lúc deploy)
initialize()