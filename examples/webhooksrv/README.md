```
curl --location --request POST 'http://localhost:8000/api/v1/users' \
--header 'Content-Type: application/json' \
--header 'X-API-KEY: secret' \
--data-raw '{
    "username": "giorgos"
}'
```
