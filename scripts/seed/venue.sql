-- Тестовые данные для ручной проверки/демонстрации. Безопасно гонять повторно (ON CONFLICT DO NOTHING).
-- Условно соответствует "Траттория Романо" из kitchen-api — своя БД, заказы со своим ID
-- (единственная связь — kitchen_order_id), но id товаров те же, что в kitchen-api: их
-- совпадение — не совпадение, а инвариант вебхука синхронизации (PUT .../menu/{itemId}
-- пишет в kitchen-api кэш-строку с тем же id, каким товар известен здесь).

INSERT INTO menu_items (id, name, description, price, is_available) VALUES
    ('b0000000-0000-0000-0000-000000000001', 'Пицца Маргарита', 'Томаты, моцарелла, базилик', 450.00, true),
    ('b0000000-0000-0000-0000-000000000002', 'Паста Карбонара', 'Бекон, яйцо, пармезан', 390.00, true),
    ('b0000000-0000-0000-0000-000000000003', 'Тирамису', 'Классический десерт', 280.00, false)
ON CONFLICT (id) DO NOTHING;

-- Соответствует заказу d0000000-0000-0000-0000-000000000001 из kitchen-api (там уже "delivered")
INSERT INTO orders (id, kitchen_order_id, status, created_at, updated_at) VALUES
    ('d1000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 'ready', now() - interval '2 hours', now() - interval '60 minutes')
ON CONFLICT (id) DO NOTHING;

INSERT INTO order_items (id, order_id, menu_item_id, name_snapshot, price_snapshot, quantity) VALUES
    ('e1000000-0000-0000-0000-000000000001', 'd1000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'Пицца Маргарита', 450.00, 2)
ON CONFLICT (id) DO NOTHING;
