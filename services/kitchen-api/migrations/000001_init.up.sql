CREATE TABLE venues (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    api_base_url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE venue_categories (
    venue_id UUID NOT NULL REFERENCES venues (id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    PRIMARY KEY (venue_id, category_id)
);

CREATE INDEX idx_venue_categories_category_id ON venue_categories (category_id);

CREATE TABLE menu_items (
    id UUID PRIMARY KEY,
    venue_id UUID NOT NULL REFERENCES venues (id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    price NUMERIC(10, 2) NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT true,
    removed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (venue_id, external_id)
);

CREATE INDEX idx_menu_items_venue_id ON menu_items (venue_id);

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    venue_id UUID NOT NULL REFERENCES venues (id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN (
        'created', 'accepted', 'rejected', 'cancelled_by_user',
        'cooking', 'cancelled_by_venue', 'ready', 'delivering', 'delivered'
    )),
    customer_name TEXT NOT NULL,
    customer_phone TEXT NOT NULL,
    delivery_address TEXT NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_venue_id ON orders (venue_id);

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

CREATE TABLE order_status_history (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_status_history_order_id ON order_status_history (order_id);
