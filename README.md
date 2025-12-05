
# Wipecord

Wipecord allows you to wipe your messages in a DM of your choice.

> [!IMPORTANT]
> Do not use Discord concurrently with Wipecord. This includes actions like sending messages, adding friends, or interacting with other elements on Discord. Doing so may increase the likelihood of hitting rate limits and, in some cases, could result in being temporarily banned from Discord's API.
  

## How to setup (Manually)

  

1. Clone this repository

```
git clone https://github.com/Lz4Lz/wipecord

cd wipecord
```

2. Run

```
go run cmd/main.go
```

  

## Usage

  

Simply put your authentication token in the login menu, and choose a DM. You can search DMS by typing in the users name or ID.

  

## How do i get my Discord Authentication Token?

>  [!CAUTION]
> Do not share this token with anyone, as it gives full access to your discord account!

  

1. Open Discord in your web browser and open developer tools by pressing F12.

2. Go to the network tab.

3. Click on anything you see there, if you dont see anything then click on a DM and you should see a network request.

4. Scroll down until you see the request headers section, find the ```Authorization``` key/header and copy its value.

  

## Improvements

Will try to fix ratelimit issues in the future, without slowing down the deletion process.

If you want to increase the speed, edit the variable ```deleteDelay``` in internal/discord/purge.go. Note that this will increase ratelimits, and will eventually throttle you.

  

## Contributing

If you have any improvements please feel free to submit a pull request or open an issue. Specifically the TUI, as this is my first TUI project. I am well aware of the poor error handling in the tui, which i will work on in the future.
