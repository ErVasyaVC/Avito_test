CREATE TABLE menu_items (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    price NUMERIC(10, 2) NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT true,
    removed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    kitchen_order_id UUID NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN (
        'accepted', 'rejected', 'cooking', 'ready',
        'cancelled_by_venue', 'cancelled_by_user'
    )),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    menu_item_id UUID NOT NULL REFERENCES menu_items (id) ON DELETE RESTRICT,
    name_snapshot TEXT NOT NULL,
    price_snapshot NUMERIC(10, 2) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);
CREATE INDEX idx_order_items_menu_item_id ON order_items (menu_item_id);
