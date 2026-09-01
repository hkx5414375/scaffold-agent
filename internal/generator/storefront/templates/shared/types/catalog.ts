export interface CatalogProduct {
  id: string;
  sku: string;
  name: string;
  description?: string;
  price_minor: string;
  currency: string;
  status: "active";
  version: string;
  created_at: string;
  updated_at: string;
}

export interface CatalogPage {
  items: CatalogProduct[];
  next_cursor?: string;
}
