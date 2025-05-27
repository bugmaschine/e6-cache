# e6-cache

**e6-cache** is a locally hosted caching proxy for e621 (or anything that's api compatible), designed to passively archive and cache content you browse.

## Why?

Because:

* You don’t remember that *one* post with the perfect lighting and suspiciously specific tags
* **You want your own archive. Your own CDN. Your own sin-server.**
* You have too much storage space
* Your Internet connection is too slow (the only legitimate use case)
* You worry about e621 going down, or posts being removed.

## What it does

* Acts as a proxy, redirecting all requests through e6-cache, and then to your chosen instance
* Transparently caches every post you view (metadata + media)
* Stores it in a local PostgreSQL database
* Saves media files in your own S3-compatible bucket

## Dev Setup

### Start DB and S3 Storage
```bash
git clone https://github.com/bugmaschine/e6-cache.git
cd e6-cache/dev
docker compose up -d
```

Development is recommended to be done in Visual Studio Code, the launch.json file is already configured for this.

You get:

* PostgreSQL database (`localhost:5432`, user: `dev`, pass: `devpass`)
* MinIO S3 storage (`localhost:9000`, user/pass: `minioadmin` ui: `http://localhost:9010`)

## Production Setup / Server Setup

> [!WARNING]  
> Don't just copy the `docker-compose.yml` file, as other files are required for the inital setup to work.

> [!IMPORTANT]
> Make sure to set the environment variables in the `docker-compose.yml` file.

```bash
git clone https://github.com/bugmaschine/e6-cache.git
cd e6-cache
docker compose up -d
```

After the container is running, you can access the API at `http://localhost:8080`, and set it as your e621 instance in your Client of choice.

## Client Setup

For most users i recommend using [e1547](https://github.com/clragon/e1547) as it has built-in support for custom instances.

### e1547 Setup
> [!IMPORTANT]
> You need to have an public URL for the API to work with e1547, as it requires https.

https://github.com/user-attachments/assets/d2304e64-0c08-4065-bd55-aaa24d13727e

### The Wolf's Stash Setup
For The Wolf's Stash, it just reports the host not being supported.

### Other Clients
Feel free to open a PR to add documentation for other clients.

## Speed Comparison (Speed depends on your internet, and database speed)

### Image 1:
- **Normal e621 image load:** 2.859s
- **Cached image load:** 1.585s


### Image 2:
- **Normal e621 image load:** 8.487s
- **Cached image load:** 1.217s

*Tests done using Firecamp*

## Planned Features

* Proxy Mode (it act's like a proxy and redirects all e621 requests to e6-cache)
* Firefox Extension (to make it easier to use by replacing all e621 links with e6-cache links)
* Offline Mode (to use the cache without internet connection, like a local e621)
* Website (basically a mirror of e621, but with the cache enabled)

## Architecture

* Every API request you make goes through `e6-cache`
* We pull the data from the API and store it in our database. And then we serve it to you
* If there are images, we create special proxy links
* If the user or a client accesses the proxy link, we check if the image is already cached, and if not, we download it and store it in the S3 bucket

## Ethical & Legal Notice

This project **does not** scrape, spider, or hammer the API. It only caches what *you* manually request.
You, as the operator, are fully responsible for your usage. This just hands you a shovel. What you dig up is on you.

## Contributing

Feel free to open an issue or a pull request. I don't have any specific guidelines for contributing, just be nice and respectful.

## Why?

Idk. I just wanted to play with Go, Docker and wanted to make something (relatively) useful. It's also my first published project on GitHub.
