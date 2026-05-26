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

# Async reference: index orders before the referenced user exists.
resource "typesense_collection" "orders_eventual" {
  name = "orders_eventual"

  fields {
    name = "order_pk"
    type = "string"
  }

  fields {
    name            = "user_pk"
    type            = "string"
    reference       = "${typesense_collection.users.name}.user_pk"
    async_reference = true
  }
}

# Auto-embedding with GCP Vertex AI authenticated via a service account.
# The service_account block is the cleanest auth path for managed embedders
# (no refresh-token rotation) and is the way to wire LOINC/SNOMED-style
# Vertex embedders that the SDK previously couldn't express.
resource "typesense_collection" "loinc_terms" {
  name = "loinc_terms"

  fields {
    name = "display"
    type = "string"
  }

  fields {
    name    = "embedding"
    type    = "float[]"
    num_dim = 768

    embed {
      from = ["display"]
      model_config {
        model_name = "gcp/text-embedding-005"
        project_id = "my-gcp-project"
        region     = "us-central1"

        service_account {
          client_email = "vertex-embedder@my-gcp-project.iam.gserviceaccount.com"
          private_key  = file("${path.module}/vertex-sa-private-key.pem")
          # token_uri defaults to https://oauth2.googleapis.com/token
        }
      }
    }
  }
}
