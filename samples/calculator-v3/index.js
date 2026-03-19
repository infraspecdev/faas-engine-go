module.exports = async function (event) {
    console.log("Received event:", event);

    const { operation, a, b } = event;

    if (typeof a !== 'number' || typeof b !== 'number') {
        console.error("Invalid input:", { a, b });
        throw new Error("Both 'a' and 'b' must be numbers.");
    }

    console.log(`\nPerforming operation: ${a} ${operation} ${b}`);

    let result;

    switch (operation) {
        case '+':
            result = a + b;
            break;
        case '-':
            result = a - b;
            break;
        case '*':
            result = a * b;
            break;
        case '/':
            result = a / b;
            break;
        default:
            console.error("Unsupported operation:", operation);
            throw new Error(`Unsupported operation: ${operation}`);
    }

    console.log("\nResult:", result);

    return result;
};