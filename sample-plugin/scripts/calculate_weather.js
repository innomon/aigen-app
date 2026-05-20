// Sample JS script for weather-plugin
log("Starting weather calculation for " + args.location);

const apiKey = getSecret("WEATHER_API_KEY");
if (!apiKey) {
    log("Error: WEATHER_API_KEY not found in vault");
} else {
    log("API Key retrieved successfully (masked)");
}

const response = fetch("https://api.weatherapi.com/v1/current.json?q=" + args.location, {
    method: "GET"
});

log("Weather API Response: " + JSON.stringify(response));

const result = {
    status: "success",
    location: args.location,
    temperature: 22,
    condition: "Sunny"
};

result; // Return value
