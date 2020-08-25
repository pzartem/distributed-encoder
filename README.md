# distributed-encoder

## Notes

Workers connect to the server using HTTP + long-polling to listen for jobs, accept the stream 
and then start a result request to the server to stream the result back.

I've chosen HTTP to keep it simple, and it's a solid solution for the file transfer (S3) + allows compression if it's needed

Ideally for production, I would consider picking up a message broker + (HTTP/GPRC any other protocol) for a file streaming, also GRPC and Websockets are possible (if you want to build your own protocol :D)

Data processing is implemented using ffmpeg and unix pipe, there is no file io in worker at all.

Orchestration is implemented using docker-compose and `--scale` flag

Possible issue as far as it's using volumes, there could be no read rights on result files for the current user

Missing features, I prefer to implement:
- graceful shutdown
- retries on fail

## How to run

```shell script
make compose-build

// will run docker compose with worker scale 4 
make compose-run 
```

To trigger a request
```shell script
curl --location --request POST 'localhost:1111/work/trigger' \
--header 'Content-Type: application/json' \
--data-raw '{
    "tiles": 16,
    "width": 7680,
    "height": 3840,
    "filePath": "/mnt/videos/LetinVR_test_1.mp4"
}'
```