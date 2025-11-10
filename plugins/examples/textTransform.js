// Text Transform Node - Example JavaScript Plugin
// This plugin demonstrates how to create a custom node that transforms text data

module.exports = {
    // Node description
    description: {
        displayName: "Text Transform",
        name: "textTransform",
        description: "Transform text data with various operations",
        category: "transform",
        icon: "fa:font",
        inputs: 1,
        outputs: 1,
        properties: [
            {
                displayName: "Operation",
                name: "operation",
                type: "options",
                default: "uppercase",
                options: [
                    { name: "Uppercase", value: "uppercase" },
                    { name: "Lowercase", value: "lowercase" },
                    { name: "Reverse", value: "reverse" },
                    { name: "Remove Spaces", value: "removeSpaces" },
                    { name: "Add Prefix", value: "addPrefix" },
                    { name: "Add Suffix", value: "addSuffix" }
                ]
            },
            {
                displayName: "Field Name",
                name: "fieldName",
                type: "string",
                default: "text",
                description: "The field to transform"
            },
            {
                displayName: "Prefix",
                name: "prefix",
                type: "string",
                default: "",
                displayOptions: {
                    show: {
                        operation: ["addPrefix"]
                    }
                }
            },
            {
                displayName: "Suffix",
                name: "suffix",
                type: "string",
                default: "",
                displayOptions: {
                    show: {
                        operation: ["addSuffix"]
                    }
                }
            }
        ]
    },

    // Execute function - this is where the node logic goes
    execute: function(inputData, parameters) {
        console.log("Executing Text Transform node");
        console.log("Operation:", parameters.operation);
        console.log("Field name:", parameters.fieldName);

        var outputData = [];

        // Process each input item
        for (var i = 0; i < inputData.length; i++) {
            var item = inputData[i];
            var json = item.json || {};
            var fieldName = parameters.fieldName || "text";
            var value = json[fieldName];

            if (typeof value !== "string") {
                console.warn("Field '" + fieldName + "' is not a string, skipping");
                outputData.push(item);
                continue;
            }

            var transformedValue = value;

            // Apply transformation based on operation
            switch (parameters.operation) {
                case "uppercase":
                    transformedValue = value.toUpperCase();
                    break;

                case "lowercase":
                    transformedValue = value.toLowerCase();
                    break;

                case "reverse":
                    transformedValue = value.split("").reverse().join("");
                    break;

                case "removeSpaces":
                    transformedValue = value.replace(/\s+/g, "");
                    break;

                case "addPrefix":
                    var prefix = parameters.prefix || "";
                    transformedValue = prefix + value;
                    break;

                case "addSuffix":
                    var suffix = parameters.suffix || "";
                    transformedValue = value + suffix;
                    break;

                default:
                    console.warn("Unknown operation:", parameters.operation);
            }

            // Create output item with transformed value
            var outputItem = {
                json: {}
            };

            // Copy all fields from input
            for (var key in json) {
                outputItem.json[key] = json[key];
            }

            // Update the transformed field
            outputItem.json[fieldName] = transformedValue;

            // Copy binary data if present
            if (item.binary) {
                outputItem.binary = item.binary;
            }

            outputData.push(outputItem);
        }

        console.log("Processed " + outputData.length + " items");
        return outputData;
    },

    // Validate parameters (optional)
    validate: function(parameters) {
        if (!parameters.fieldName) {
            return "Field name is required";
        }
        return null; // null means validation passed
    }
};
