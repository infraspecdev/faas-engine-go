module.exports = async function (event) {
    const sleep = (ms) => new Promise(res => setTimeout(res, ms));

    console.log("Received event:", event);
    await sleep(500);

    const { operation, a, b } = event;

    if (typeof a !== 'number' || typeof b !== 'number') {
        console.error("Invalid input:", { a, b });
        await sleep(500);
        throw new Error("Both 'a' and 'b' must be numbers.");
    }

    console.log(`Performing operation: ${a} ${operation} ${b}`);
    await sleep(700);

    let result;

    switch (operation) {
        case '+':
            console.log("Adding numbers...");
            await sleep(600);
            result = a + b;
            break;

        case '-':
            console.log("Subtracting numbers...");
            await sleep(600);
            result = a - b;
            break;

        case '*':
            console.log("Multiplying numbers...");
            await sleep(600);
            result = a * b;
            break;

        case '/':
            console.log("Dividing numbers...");
            await sleep(600);
            result = a / b;
            break;

        default:
            console.error("Unsupported operation:", operation);
            await sleep(500);
            throw new Error(`Unsupported operation: ${operation}`);
    }

    console.log("Calculation done...");
    await sleep(500);

    console.log("Result:", result);
    await sleep(500);

    return result;
};