
# Hermeshooks - A web hook scheduler


Hermeshooks is an application that helps you to schedule webhook execution.

It exposes a REST API that helps you to schedule the execution of webhooks
at a specified time. 

It is useful when you have an application that needs to do some action at 
specific time.

### Some possible use cases

- you need to sent a notification to a user of your app at a specific time
   (could be a reminder, a push notification, or whatever you need to execute)
- you have a blog engine and you want to publish a post at a specific time
- you need to do some action in your application at a specific time
- sent emails to users to review your products N days after ordering
- add your use case here ... 

### Why not use cron or at commands

The above tasks can be done also by `cron` or `at` in a Linux system. 
Although, sometimes it's not convenient and you need to do some extra work.

Additionally, Hermeshooks can be invonked by evey technology you are using. 
You just need to make an HTTP request and ofcourse create the webhook handler
which contains your logic.

The important here is that you do not need to do anything extra apart from that.


## Quickstart 

### via rapidAPI

Make an account to rapidAPI and use the Free tier to try it out


If you want to run this on production and you need to exceed the free tier 
then you should:

   - upgrade to a paid version 

   or 

   -  deploy it yourself. 

I recommend first that you try the Free version from rapidApi.

If you are satisfied and you wish to use it on production
you can upgrade to a paid version.

Alternatively, you can host the service  yourself (see following section). 



### run it your self

I have provided a `docker-compose.yaml` file with some configuration.
you can use that as a basis for your deployment or to try it locally.


```
docker-compose up -d
```

The above starts the web server and one worker. 
 

Then create an API key by registering a user. 

```
curl --location --request POST 'http://localhost:8000/api/v1/users' \
--header 'Content-Type: application/json' \
--header 'X-API-KEY: secret' \
--data-raw '{
    "username": "giorgos"
}'
```

Note down the `apiKey` returned. 
You are going to use it for your requests.

See folder `examples` for an example of usages. The example 
contains 2 endpoints one to process the webhooks and one to make a POST
request to the service. Please adjust the variables to match your machine


If you are going to deploy to production do not forget to 
change the `INTERNAL_API_KEY` environment variable at least. 
Default value is `secret` which is not so secret. 
