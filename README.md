# Velux-Nibe

Use VELUX ACTIVE sensors as smart thermostats in NIBE Uplink.

## Purpose

[NIBE Uplink](https://www.nibeuplink.com) gives the option of connecting your heat pump to a smart home system and making your heating system even smarter. Connect the thermostats in your smart home system with NIBE Uplink for optimised control of your indoor climate.

Unfortunately, the list of compatible systems is tiny. As of March 2021, only some thermostats from Niko are supported along with an integration based on IFTTT. This project extends this list by integrating [VELUX ACTIVE](https://www.velux.com/active) sensors which you may be using to control your windows and shades.

## Usage

### 1. Get NIBE API credentials

The tool uses the [NIBE Uplink API](http://api.nibeuplink.com) to communicate with your heat pump and requires
API credentials to do so. Use your existing NIBE account to create an application at https://api.nibeuplink.com/Applications.
As callback URL, you can use `https://www.marshflattsfarm.org.uk/nibeuplink/oauth2callback/index.php` or host a similar script as in https://www.marshflattsfarm.org.uk/wordpress/?page_id=3480 on web servers of your choice. The application name is arbitrary, but what you enter in this field will be visible in the portal as the smart thermostat source.

Note your NIBE system ID which is visible in the URL after logging in to NIBE Uplink, e.g. in `https://www.nibeuplink.com/System/${ID}/Status/Overview`.

### 2. First run

Equipped with the client ID and secret from step 1 and your VELUX ACTIVE credentials, run the tool once to generate an access token (this step needs to be done only once):

```
go get github.com/IngmarStein/velux-nibe
./velux-nibe -velux-user xxx -velux-password xxx -nibe-system xxx -nibe-client-id xxx -nibe-client-secret xxx -nibe-callback xxx
```

You will be asked to open an URL in your browser and solve a captcha. Verify that the `state` parameter is set to `state-token` and enter the resulting code in `velux-nibe`. If successful, this will save an OAuth2 access token to the default location `nibe-token.json`.

### 3. Run `velux-nibe`

There are several options to run the tool on a variety of machines, including Raspberry Pi or common NAS hardware.

#### Native

```
go get github.com/IngmarStein/velux-nibe
VELUX_USERNAME=xxx VELUX_PASSWORD=xxx NIBE_CLIENT_ID=xxx NIBE_CLIENT_SECRET=xxx NIBE_CALLBACK_URL=xxx NIBE_SYSTEM_ID=xxx NIBE_TOKEN=nibe-token.json velux-nibe -targetTemp 210 -interval 60
```

#### In a container

```
docker run -v nibe-token.json:/nibe-token.json --env VELUX_USERNAME=xxx --env VELUX_PASSWORD=xxx --env NIBE_CLIENT_ID=xxx --env NIBE_CLIENT_SECRET=xxx --env NIBE_CALLBACK_URL=xxx --env NIBE_SYSTEM_ID=xxx ingmarstein/velux-nibe -targetTemp 210 -interval 60
```

Alternatively, use the included `docker-compose.yml` file as a template if you prefer to use Docker Compose.

### 4. Enable Smart Home mode

Once `velux-nibe` is running, it is polling your thermostats in the defined interval and submits the current values (as well as the specified target temperature) to NIBE Uplink. You can verify the data in the portal in the section "My Systems > System > Smart Home > Thermostats".

If you are happy with the results, don't forget to enable "smart home" mode in "My Systems > System > Manage > heat pump > plus functions > smart home" so that the heat pump actually uses the indoor temperatures to optimize operations.
