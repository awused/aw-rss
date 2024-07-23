# AW-RSS

An RSS/Atom aggregator with a web frontend.

## Running Locally

`cargo install --git github.com/awused/aw-rss --locked`

Copy `aw-rss.toml.sample` to `~/.config/aw-rss/aw-rss.toml` or `~/.aw-rss.toml`
and fill it out according to the instructions.

Run `aw-rss` and navigate to `http://localhost:9092` or the port you configured
to access the application. The process will shut down cleanly if killed with
ctrl-C/SIGINT.

## Remote Access

Aw-RSS does not handle any kind of security, authentication, or authorization so
it is not safe to expose to the internet. At the minimum you'll need a reverse
proxy like nginx with HTTP basic authentication to protect it.

If setting up some form of reverse proxy, you can more efficiently serve the
static files by configuring your webserver to directly serve the [dist](dist/)
directory while falling back to index.html. Example nginx config:

```
location /api {
    proxy_pass http://localhost:9092;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}

location /index.html {
    alias /path/to/aw-rss/dist/index.html;
}

location / {
    alias /storage/src/awused/aw-rss/dist/;
    try_files $uri /index.html;
    expires max;
}
```

<!-- # Cloudflare -->
<!--  -->
<!-- I include some limited workarounds for cloudflare protectected feeds. I update this as necessary, it is currently using: -->
<!--  -->
<!-- * python3 -->
<!-- * [cloudscraper](https://github.com/venomous/cloudscraper) -->
<!--  -->
<!-- As a safeguard you'll have to use HTTPS and whitelist individual hosts in the config file to avoid running javascript you don't minimally trust. -->

## External Commands

I have support for running external commands that generate RSS or atom feeds on
stdout in place of calling HTTP servers. Use this if you want to write your own
scraper for a website that does not provide feeds.

You cannot add these using the web frontend, you must use the sqlite3 CLI to do it.
In place of a url place a shell command prepended by an exclamation point.

Example: `INSERT INTO feeds(url) VALUES('!my-command arg1 arg2 arg3');`

If the feed produced has an `aw-rss:etag` extension, that will be passed back to
the program using an appended `--etag` parameter. If the program prints
"not modified" as its only output and exits normally it will be handled as an
HTTP not modified response.

## Local Development

You'll have to edit proxy.conf.json to match the server configuration.

Run `docker-compose run --rm build` to build and compress the frontend.

Run `docker-compose run --rm dev` for an angular dev server.
Navigate to `http://localhost:4200/` with the backend already running.

## Shortcuts

Shortcut | Action
---------| ----------
`Middle Click` | Opens an item while marking it as read at the same time.
`Right Click` | Marks an unread item as read or a read item as unread.
`R` | Refresh.
`N` | Open the add new Feed/Category dialog.
`A` | Open the admin page.

## Why

I built this because I did not like any of the RSS readers, free or paid,
I tried after Google Reader died.

Tiny Tiny RSS is the closest thing and I likely would have used it but I
did not want to run PHP on my server. Since starting it I've been able to
add niche features and workaround for broken feeds that wouldn't be appropriate
in a large and widely used project like tt-rss.

