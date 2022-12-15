# wcaptcha-api
Backend API of wCaptcha

[中文说明点此](https://github.com/wcaptcha/wcaptcha-api/blob/master/README.zh.md)

## Private Deployment

wCaptcha provides official services, so if you don't want to deploy it yourself, you can go directly to using [official services](https://wcaptcha.pingflash.com/).

If you want to deploy privately, follow the steps below.

### General Deployment

Before installing, make sure you have go installed

Build from source
```shell
git clone https://github.com/wcaptcha/wcaptcha-api
cd wcaptcha-api
go build
```

You can also download the pre-compiled binaries at the [Release page](https://github.com/wcaptcha/wcaptcha-api/releases)

Next, create a configuration file named `.env`, you can just `cp .env.example .env` then edit it.

Once the configuration is set, you can start the service.

```shell
. /wcaptcha
```

Now let's create a site.
```shell
curl -X POST localhost:8090/site/create
```
The result is returned in JSON format, containing `api_key` and `api_secret`, which is the key of your site.

To modify the difficulty (aka client proofing time), you can execute.
```shell
curl -X post localhost:8090/site/update --data "api_secret=YOUR_API_SECRET&hardness=HARDNESS
```
Where HARDNESS is a number, the default HARDNESS for a site is `4194303` (`2^22-1`), you can set any number.



### Deploy to Vercel

You can also deploy the service to vercel. Please be noticed `file` storage is not available while deploying to vercel, because vercel is a serverless and the filesystem is not persist.

First install the vercel cli tool, 

```shell
npm i -g vercel
```

Then run:

```shell
cp .env .env.production
./vercel-deploy.sh
```

then follow the instructions to finish deployment.


## Set wcaptcha-js To Use Private Deployed Service

```javascript
w = new wcaptcha(API_KEY)
w.setEndpoint("https://your-deployed-service.com/")

// Then use wcaptcha as usual
// w.bind("any-selector")
```
