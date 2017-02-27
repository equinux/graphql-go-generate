# graphql-go-generate
Extract and generate GraphQL schema definitions from Go structs.

## What?
Create [graphql-go](https://github.com/graphql-go/graphql) object definitions for all `struct` types with tagged fields from any package.

## How?
```
go install github.com/equinux/graphql-go-generate
graphql-go-generate --tag=json github.com/square/go-jose
```

## Why?
This gives you a quick starting point, if you have a lot of models to put in your schema. 

## Enjoy
🍻
