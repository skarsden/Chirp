Chirp
-----
A small project I did to learn about the workings of http servers, requests, responses, and webhooks. Requires Go, Postgres, Goose, and SQLC to function.

Usage
-----
Start the server to open it up to http requests (I used 'go build -o out && ./out' in my terminal during development)

Use whichever http request software you prefer (REST, Thunder, Postman, etc.) to send you http requests to the endpoints in the main.go file. Proper request parameters can be found at the top of the associated handler functions via the 'reqParam' structs if they require them.

the 'api/chirps' endpoint optionally takes additional url params: 

author_id - a uuid representing an existing user that will cause only chirps belonging to that user to be returned

sort - changes the sorted order of returned chirps, either 'asc' or 'desc'. 'asc is the default.
