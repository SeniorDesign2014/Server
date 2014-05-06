## Server

The back end server for the Bike Theft Tracker.  This server runs on Google App Engine.  This code can also be run locally with the [Google App Engine SDK](https://developers.google.com/appengine/downloads).

The purpose of this software is to interface with both the Bike Theft Tracker module on the bicycle and the user's mobile app.  The server notifies the user of bike theft with SMS, email and iOS push notifications, and the app receives the bike's most recent known location.

#### Other software:
[gotwilio](https://github.com/sfreiberg/gotwilio/blob/master/gotwilio.go) - API for Twilio by [sfreiberg](https://github.com/sfreiberg) modified for use with Google App Engine

[Anachronistic apns](https://github.com/anachronistic/apns/) - Framework for sending messages to Apple Notification Service servers for the purpose of sending push notifications.  Modified for use with Google App Engine.
