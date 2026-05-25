resource "typesense_collection" "my_collection" {
  name = "my-collection"

  fields {
    facet    = true
    index    = true
    name     = "testFiled1"
    optional = true
    type     = "string"
  }

  fields {
    facet    = true
    index    = true
    name     = "testFiled2"
    optional = true
    type     = "int32"
  }
}

# JOIN reference: link orders.user_pk to users.user_pk for query-time joins.
resource "typesense_collection" "users" {
  name = "users"

  fields {
    name = "user_pk"
    type = "string"
  }

  fields {
    name = "email"
    type = "string"
  }
}

resource "typesense_collection" "orders" {
  name = "orders"

  fields {
    name = "order_pk"
    type = "string"
  }

  fields {
    name      = "user_pk"
    type      = "string"
    reference = "${typesense_collection.users.name}.user_pk"
  }

  fields {
    name        = "amount"
    type        = "float"
    range_index = true
    sort        = true
  }

  fields {
    name     = "embedding"
    type     = "float[]"
    num_dim  = 384
    vec_dist = "cosine"
  }
}
