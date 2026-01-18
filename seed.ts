import { Database } from 'bun:sqlite'
import { mkdir, rm } from 'node:fs/promises'

type SeedConfig = {
    dbPath: string
}

function parseSeedArgs(argv: string[]): SeedConfig {
    let dbPath = 'data/seed.db'

    for (const arg of argv) {
        if (arg.startsWith('--db=')) {
            dbPath = arg.slice('--db='.length)
            continue
        }

        if (!arg.startsWith('-')) {
            dbPath = arg
        }
    }

    return { dbPath }
}

type Customer = {
    name: string
    email: string
}

type Product = {
    sku: string
    name: string
    priceCents: number
}

type OrderStatus = 'pending' | 'paid' | 'shipped' | 'cancelled'

type OrderSeed = {
    customerId: number
    status: OrderStatus
    createdAt: string
    items: Array<{ productId: number; quantity: number; unitPriceCents: number }>
}

function isoDaysAgo(daysAgo: number): string {
    const now = new Date()
    now.setUTCDate(now.getUTCDate() - daysAgo)
    return now.toISOString()
}

async function main(): Promise<void> {
    const config = parseSeedArgs(process.argv.slice(2))

    await mkdir('data', { recursive: true })

    await rm(config.dbPath, { force: true })

    const db = new Database(config.dbPath)

    db.exec('PRAGMA journal_mode = WAL')
    db.exec('PRAGMA foreign_keys = ON')

    const schemaSql = [
        'CREATE TABLE IF NOT EXISTS customers (',
        '  id INTEGER PRIMARY KEY AUTOINCREMENT,',
        '  name TEXT NOT NULL,',
        '  email TEXT NOT NULL UNIQUE,',
        '  created_at TEXT NOT NULL',
        ');',
        '',
        'CREATE TABLE IF NOT EXISTS products (',
        '  id INTEGER PRIMARY KEY AUTOINCREMENT,',
        '  sku TEXT NOT NULL UNIQUE,',
        '  name TEXT NOT NULL,',
        '  price_cents INTEGER NOT NULL',
        ');',
        '',
        'CREATE TABLE IF NOT EXISTS orders (',
        '  id INTEGER PRIMARY KEY AUTOINCREMENT,',
        '  customer_id INTEGER NOT NULL REFERENCES customers(id),',
        '  status TEXT NOT NULL,',
        '  created_at TEXT NOT NULL',
        ');',
        '',
        'CREATE TABLE IF NOT EXISTS order_items (',
        '  id INTEGER PRIMARY KEY AUTOINCREMENT,',
        '  order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,',
        '  product_id INTEGER NOT NULL REFERENCES products(id),',
        '  quantity INTEGER NOT NULL,',
        '  unit_price_cents INTEGER NOT NULL',
        ');',
        '',
        'CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);',
        'CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);',
        'CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);',
    ].join('\n')

    db.exec(schemaSql)

    const existingCustomers = db
        .query<{ count: number }, []>('SELECT COUNT(*) AS count FROM customers')
        .get()
    const shouldInsertData = (existingCustomers?.count ?? 0) === 0

    if (!shouldInsertData) {
        db.close()
        console.log(`Seed DB already has data: ${config.dbPath}`)
        return
    }

    const customers: Customer[] = []
    for (let i = 1; i <= 500; i += 1) {
        const name = `Customer ${i}`
        const email = `customer.${i}@example.com`
        customers.push({ name, email })
    }

    const products: Product[] = []
    for (let i = 1; i <= 200; i += 1) {
        const sku = `SKU-${String(i).padStart(4, '0')}`
        const name = `Product ${i}`
        const priceCents = 500 + (i % 50) * 75
        products.push({ sku, name, priceCents })
    }

    const insertCustomer = db.prepare<unknown, [string, string, string]>(
        'INSERT INTO customers (name, email, created_at) VALUES (?, ?, ?)',
    )
    const insertProduct = db.prepare<unknown, [string, string, number]>(
        'INSERT INTO products (sku, name, price_cents) VALUES (?, ?, ?)',
    )

    const insertCustomersTx = db.transaction(() => {
        for (const customer of customers) {
            insertCustomer.run(customer.name, customer.email, isoDaysAgo(30))
        }
    })

    const insertProductsTx = db.transaction(() => {
        for (const product of products) {
            insertProduct.run(product.sku, product.name, product.priceCents)
        }
    })

    insertCustomersTx()
    insertProductsTx()

    const productIdBySku = new Map<string, number>()
    const productRows = db
        .query<{ id: number; sku: string }, []>('SELECT id, sku FROM products ORDER BY id')
        .all()
    for (const row of productRows) {
        productIdBySku.set(row.sku, row.id)
    }

    const customerIdByEmail = new Map<string, number>()
    const customerRows = db
        .query<{ id: number; email: string }, []>('SELECT id, email FROM customers ORDER BY id')
        .all()
    for (const row of customerRows) {
        customerIdByEmail.set(row.email, row.id)
    }

    function getCustomerId(index: number): number {
        const email = `customer.${index}@example.com`
        const value = customerIdByEmail.get(email)
        if (value === undefined) {
            throw new Error(`Missing customer id for ${email}`)
        }
        return value
    }

    function getProductId(index: number): number {
        const sku = `SKU-${String(index).padStart(4, '0')}`
        const value = productIdBySku.get(sku)
        if (value === undefined) {
            throw new Error(`Missing product id for ${sku}`)
        }
        return value
    }

    const orders: OrderSeed[] = []
    for (let i = 1; i <= 5000; i += 1) {
        const customerId = getCustomerId(((i - 1) % customers.length) + 1)
        const statusOptions: OrderStatus[] = ['pending', 'paid', 'shipped', 'cancelled']
        const status = statusOptions[i % statusOptions.length] ?? 'pending'
        const createdAt = isoDaysAgo(i % 45)
        const items: OrderSeed['items'] = []
        const itemCount = 1 + (i % 4)

        for (let j = 0; j < itemCount; j += 1) {
            const productIndex = ((i + j) % products.length) + 1
            const productId = getProductId(productIndex)
            const quantity = 1 + ((i + j) % 3)
            const product = products[productIndex - 1]
            if (!product) {
                continue
            }

            items.push({
                productId,
                quantity,
                unitPriceCents: product.priceCents,
            })
        }

        orders.push({ customerId, status, createdAt, items })
    }

    const insertOrder = db.prepare<unknown, [number, string, string]>(
        'INSERT INTO orders (customer_id, status, created_at) VALUES (?, ?, ?)',
    )
    const insertOrderItem = db.prepare<unknown, [number, number, number, number]>(
        'INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents) VALUES (?, ?, ?, ?)',
    )

    const insertOrdersTx = db.transaction(() => {
        for (const order of orders) {
            const result = insertOrder.run(order.customerId, order.status, order.createdAt)

            const orderId = result.lastInsertRowid
            let orderIdNumber: number

            if (typeof orderId === 'bigint') {
                orderIdNumber = Number(orderId)
            } else {
                orderIdNumber = orderId
            }

            for (const item of order.items) {
                insertOrderItem.run(
                    orderIdNumber,
                    item.productId,
                    item.quantity,
                    item.unitPriceCents,
                )
            }
        }
    })

    insertOrdersTx()

    db.close()
    console.log(`Seeded DB: ${config.dbPath}`)
}

await main()
