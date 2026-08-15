# **UPDATED FORK**

As the original repository did not receive updates nor PRs for a couple of years, we will use this fork to address some issues and upgrade its capabilities.
Read the changes carefully.

## New capabilities

### Sygnia

* Upgraded dependencies (golang.org/x/crypto) to address Autocert fails to sign certificate automatically with Let's encrypt.
* Added capability to upload via command line utilities. A new button added to the main page UI, where you can get an example cURL command to use for upload:
<p align="center">
  <img alt="uploadCurl" src="media/upload-via-curl.gif" height="500" />
</p>

* When uploading via command line utility you can specify the mime-type via special header **x-pwndrop-content-type**:
```
curl -X POST -H "x-pwndrop-content-type: application/javascript" -H "Authorization: [redacted]" -F "file=@Malicious.exe"  https://<your-server>/api/v1/files
```

### tooothl3ss

* Folders — organize files into folders (shared across all users, visual grouping only)
* File comments — add a short note to any file, visible to everyone in the panel
* Download limit — set a max download count per file (0 = unlimited); file auto-blocks after limit is reached
* Download log — per-file log of every download: IP address, user-agent, timestamp; live auto-refresh every 5s in the panel
* API token upload — upload files via curl without a browser session; token is auto-generated and shown via the curl button in the panel
* No proxy caching — Cache-Control: no-store is set on all served files to prevent CDN/proxy from caching disabled files

### Original Features

* Upload and immediately share multiple files using drag & drop
* Make files available or unavailable for download with a single click
* Set up custom download URLs without playing with directory structure
* Facade files — serve a decoy file instead of the original on demand
* Automatic redirects to spoof file extension in a shared link
* Change MIME type of served file
* Serve files over HTTP, HTTPS and WebDAV
* Admin panel behind a custom secret URL path
* Auto-generated HTTPS certificates via Let's Encrypt


## License

**pwndrop** is made by Kuba Gretzky ([@mrgretzky](https://twitter.com/mrgretzky)) and it's released under GPL3 license.
