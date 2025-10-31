-- Contract Counter (Stateful)

function initialize()
    db_put("counter", 0)
end

function increment()
    local current_val = db_get("counter")
    local new_val = tonumber(current_val) + 1
    db_put("counter", new_val)
end

initialize()