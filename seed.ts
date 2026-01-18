import { Database } from "bun:sqlite";
import { mkdir, rm } from "node:fs/promises";

type SeedConfig = {
    dbPath: string;
    reset: boolean;
};

function parseSeedArgs(argv: string[]): SeedConfig {
    let dbPath = "data/seed.db";
    let reset = true;

    for (const arg of argv) {
        if (arg === "--no-reset") {
            reset = false;
            continue;
        }

        if (arg.startsWith("--db=")) {
            dbPath = arg.slice("--db=".length);
            continue;
        }

        if (!arg.startsWith("-")) {
            dbPath = arg;
        }
    }

    return { dbPath, reset };
}

type Customer = {
    name: string;
    email: string;
};

type Product = {
    sku: string;
    name: string;
    priceCents: number;
};

type OrderStatus = "pending" | "paid" | "shipped" | "cancelled";

type OrderSeed = {
    customerId: number;
    status: OrderStatus;
    createdAt: string;
    items: Array<{ productId: number; quantity: number; unitPriceCents: number }>;
};

function isoDaysAgo(daysAgo: number): string {
    const now = new Date();
    now.setUTCDate(now.getUTCDate() - daysAgo);
    return now.toISOString();
}

async function main(): Promise<void> {
    const config = parseSeedArgs(process.argv.slice(2));

    await mkdir("data", { recursive: true });

    if (config.reset) {
        await rm(config.dbPath, { force: true });
    }

    const db = new Database(config.dbPath);

    db.exec("PRAGMA journal_mode = WAL");
    db.exec("PRAGMA foreign_keys = ON");

    const schemaSql = [
        "CREATE TABLE IF NOT EXISTS customers (",
        "  id INTEGER PRIMARY KEY AUTOINCREMENT,",
        "  name TEXT NOT NULL,",
        "  email TEXT NOT NULL UNIQUE,",
        "  created_at TEXT NOT NULL",
        ");",
        "",
        "CREATE TABLE IF NOT EXISTS products (",
        "  id INTEGER PRIMARY KEY AUTOINCREMENT,",
        "  sku TEXT NOT NULL UNIQUE,",
        "  name TEXT NOT NULL,",
        "  price_cents INTEGER NOT NULL",
        ");",
        "",
        "CREATE TABLE IF NOT EXISTS orders (",
        "  id INTEGER PRIMARY KEY AUTOINCREMENT,",
        "  customer_id INTEGER NOT NULL REFERENCES customers(id),",
        "  status TEXT NOT NULL,",
        "  created_at TEXT NOT NULL",
        ");",
        "",
        "CREATE TABLE IF NOT EXISTS order_items (",
        "  id INTEGER PRIMARY KEY AUTOINCREMENT,",
        "  order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,",
        "  product_id INTEGER NOT NULL REFERENCES products(id),",
        "  quantity INTEGER NOT NULL,",
        "  unit_price_cents INTEGER NOT NULL",
        ");",
        "",
        "CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);",
        "CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);",
        "CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);",
    ].join("\n");

    db.exec(schemaSql);

    const existingCustomers = db.query<{ count: number }, []>("SELECT COUNT(*) AS count FROM customers").get();
    const shouldInsertData = (existingCustomers?.count ?? 0) === 0;

    if (!shouldInsertData) {
        db.close();
        console.log(`Seed DB already has data: ${config.dbPath}`);
        return;
    }

    const customers: Customer[] = [
        { name: "Ava Chen", email: "ava.chen@example.com" },
        { name: "Mateo Rivera", email: "mateo.rivera@example.com" },
        { name: "Noah Patel", email: "noah.patel@example.com" },
        { name: "Sofia Garcia", email: "sofia.garcia@example.com" },
        { name: "Liam Nguyen", email: "liam.nguyen@example.com" },
        { name: "Emma Johnson", email: "emma.johnson@example.com" },
        { name: "Amir Ali", email: "amir.ali@example.com" },
        { name: "Mia Rossi", email: "mia.rossi@example.com" },
    ];

    const products: Product[] = [
        { sku: "SKU-TEA-001", name: "Jasmine Green Tea", priceCents: 1299 },
        { sku: "SKU-COF-001", name: "Ethiopian Coffee Beans", priceCents: 1799 },
        { sku: "SKU-MUG-001", name: "Ceramic Mug", priceCents: 1599 },
        { sku: "SKU-FLT-001", name: "Reusable Filter", priceCents: 899 },
        { sku: "SKU-CHC-001", name: "Dark Chocolate Bar", priceCents: 499 },
        { sku: "SKU-HNY-001", name: "Wildflower Honey", priceCents: 1099 },
        { sku: "SKU-CRM-001", name: "Oat Creamer", priceCents: 699 },
        { sku: "SKU-BIS-001", name: "Butter Biscuits", priceCents: 549 },
        { sku: "SKU-SPN-001", name: "Espresso Spoon Set", priceCents: 1899 },
        { sku: "SKU-KET-001", name: "Mini Kettle", priceCents: 3499 },
        { sku: "SKU-BRW-001", name: "Pour-over Brewer", priceCents: 2499 },
        { sku: "SKU-NAP-001", name: "Linen Napkins", priceCents: 2199 },
    ];

    const insertCustomer = db.prepare<unknown, [string, string, string]>(
        "INSERT INTO customers (name, email, created_at) VALUES (?, ?, ?)",
    );
    const insertProduct = db.prepare<unknown, [string, string, number]>(
        "INSERT INTO products (sku, name, price_cents) VALUES (?, ?, ?)",
    );

    const insertCustomersTx = db.transaction(() => {
        for (const customer of customers) {
            insertCustomer.run(customer.name, customer.email, isoDaysAgo(30));
        }
    });

    const insertProductsTx = db.transaction(() => {
        for (const product of products) {
            insertProduct.run(product.sku, product.name, product.priceCents);
        }
    });

    insertCustomersTx();
    insertProductsTx();

    const productIdBySku = new Map<string, number>();
    const productRows = db.query<{ id: number; sku: string }, []>("SELECT id, sku FROM products ORDER BY id").all();
    for (const row of productRows) {
        productIdBySku.set(row.sku, row.id);
    }

    const customerIdByEmail = new Map<string, number>();
    const customerRows = db.query<{ id: number; email: string }, []>("SELECT id, email FROM customers ORDER BY id").all();
    for (const row of customerRows) {
        customerIdByEmail.set(row.email, row.id);
    }

    function getCustomerId(email: string): number {
        const value = customerIdByEmail.get(email);
        if (value === undefined) {
            throw new Error(`Missing customer id for ${email}`);
        }
        return value;
    }

    function getProductId(sku: string): number {
        const value = productIdBySku.get(sku);
        if (value === undefined) {
            throw new Error(`Missing product id for ${sku}`);
        }
        return value;
    }

    const orders: OrderSeed[] = [
        {
            customerId: getCustomerId("ava.chen@example.com"),
            status: "paid",
            createdAt: isoDaysAgo(7),
            items: [
                { productId: getProductId("SKU-COF-001"), quantity: 1, unitPriceCents: 1799 },
                { productId: getProductId("SKU-MUG-001"), quantity: 2, unitPriceCents: 1599 },
            ],
        },
        {
            customerId: getCustomerId("mateo.rivera@example.com"),
            status: "shipped",
            createdAt: isoDaysAgo(3),
            items: [
                { productId: getProductId("SKU-TEA-001"), quantity: 3, unitPriceCents: 1299 },
                { productId: getProductId("SKU-HNY-001"), quantity: 1, unitPriceCents: 1099 },
            ],
        },
        {
            customerId: getCustomerId("noah.patel@example.com"),
            status: "pending",
            createdAt: isoDaysAgo(1),
            items: [
                { productId: getProductId("SKU-BRW-001"), quantity: 1, unitPriceCents: 2499 },
                { productId: getProductId("SKU-FLT-001"), quantity: 2, unitPriceCents: 899 },
                { productId: getProductId("SKU-CHC-001"), quantity: 4, unitPriceCents: 499 },
            ],
        },
        {
            customerId: getCustomerId("sofia.garcia@example.com"),
            status: "cancelled",
            createdAt: isoDaysAgo(12),
            items: [
                { productId: getProductId("SKU-KET-001"), quantity: 1, unitPriceCents: 3499 },
                { productId: getProductId("SKU-SPN-001"), quantity: 1, unitPriceCents: 1899 },
            ],
        },
        {
            customerId: getCustomerId("liam.nguyen@example.com"),
            status: "paid",
            createdAt: isoDaysAgo(20),
            items: [
                { productId: getProductId("SKU-BIS-001"), quantity: 2, unitPriceCents: 549 },
                { productId: getProductId("SKU-CRM-001"), quantity: 3, unitPriceCents: 699 },
            ],
        },
        {
            customerId: getCustomerId("emma.johnson@example.com"),
            status: "shipped",
            createdAt: isoDaysAgo(16),
            items: [
                { productId: getProductId("SKU-NAP-001"), quantity: 1, unitPriceCents: 2199 },
                { productId: getProductId("SKU-MUG-001"), quantity: 1, unitPriceCents: 1599 },
            ],
        },
    ];

    const insertOrder = db.prepare<unknown, [number, string, string]>(
        "INSERT INTO orders (customer_id, status, created_at) VALUES (?, ?, ?)",
    );
    const insertOrderItem = db.prepare<unknown, [number, number, number, number]>(
        "INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents) VALUES (?, ?, ?, ?)",
    );

    const insertOrdersTx = db.transaction(() => {
        for (const order of orders) {
            const result = insertOrder.run(order.customerId, order.status, order.createdAt);

            const orderId = result.lastInsertRowid;
            let orderIdNumber: number;

            if (typeof orderId === "bigint") {
                orderIdNumber = Number(orderId);
            } else {
                orderIdNumber = orderId;
            }

            for (const item of order.items) {
                insertOrderItem.run(orderIdNumber, item.productId, item.quantity, item.unitPriceCents);
            }
        }
    });

    insertOrdersTx();

    db.close();
    console.log(`Seeded DB: ${config.dbPath}`);
}

await main();
