// Sample tool logic
(function() {
    const location = args.location || "Unknown";
    const temp = Math.floor(Math.random() * 30) + 10; // Mock temp
    
    aigen.Log("info", "Calculating weather for " + location);
    
    return {
        location: location,
        temperature: temp,
        unit: "Celsius",
        condition: "Partly Cloudy"
    };
})()
