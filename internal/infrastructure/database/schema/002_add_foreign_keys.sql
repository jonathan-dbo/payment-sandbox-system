DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_name = 'merchants'
  ) THEN
    ALTER TABLE merchants
      ADD COLUMN IF NOT EXISTS user_id TEXT;
  END IF;

  -- Backfill legacy rows when merchant id already matches user id.
  UPDATE merchants m
  SET user_id = m.id
  WHERE m.user_id IS NULL
    AND EXISTS (SELECT 1 FROM users u WHERE u.id = m.id);

  CREATE UNIQUE INDEX IF NOT EXISTS idx_merchants_user_id
    ON merchants(user_id)
    WHERE user_id IS NOT NULL;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_merchants_user_id'
  ) THEN
    ALTER TABLE merchants
      ADD CONSTRAINT fk_merchants_user_id
      FOREIGN KEY (user_id) REFERENCES users(id)
      ON UPDATE RESTRICT
      ON DELETE RESTRICT;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_invoices_merchant_id'
  ) THEN
    ALTER TABLE invoices
      ADD CONSTRAINT fk_invoices_merchant_id
      FOREIGN KEY (merchant_id) REFERENCES merchants(id)
      ON UPDATE RESTRICT
      ON DELETE RESTRICT;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_payment_intents_invoice_id'
  ) THEN
    ALTER TABLE payment_intents
      ADD CONSTRAINT fk_payment_intents_invoice_id
      FOREIGN KEY (invoice_id) REFERENCES invoices(id)
      ON UPDATE RESTRICT
      ON DELETE RESTRICT;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_refunds_invoice_id'
  ) THEN
    ALTER TABLE refunds
      ADD CONSTRAINT fk_refunds_invoice_id
      FOREIGN KEY (invoice_id) REFERENCES invoices(id)
      ON UPDATE RESTRICT
      ON DELETE RESTRICT;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_refunds_merchant_id'
  ) THEN
    ALTER TABLE refunds
      ADD CONSTRAINT fk_refunds_merchant_id
      FOREIGN KEY (merchant_id) REFERENCES merchants(id)
      ON UPDATE RESTRICT
      ON DELETE RESTRICT;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_wallets_merchant_id'
  ) THEN
    ALTER TABLE wallets
      ADD CONSTRAINT fk_wallets_merchant_id
      FOREIGN KEY (merchant_id) REFERENCES merchants(id)
      ON UPDATE RESTRICT
      ON DELETE RESTRICT;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_top_ups_merchant_id'
  ) THEN
    ALTER TABLE top_ups
      ADD CONSTRAINT fk_top_ups_merchant_id
      FOREIGN KEY (merchant_id) REFERENCES merchants(id)
      ON UPDATE RESTRICT
      ON DELETE RESTRICT;
  END IF;
END $$;

