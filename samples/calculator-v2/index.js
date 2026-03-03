module.exports = async function (event) {
    const { operation, a, b } = event;
    if (typeof a !== 'number' || typeof b !== 'number') {
        throw new Error("Both 'a' and 'b' must be numbers.");
    }

    switch (operation) {
        case '+':
            return a + b;
        case '-':
            return a - b;
        case '*':
            return a * b;
        case '/':
            return a / b;
        default:
            throw new Error(`Unsupported operation: ${operation}`);
    }
}