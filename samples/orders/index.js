module.exports = async function handler(event) {
  if (!event || typeof event !== "object") {
    throw new Error("Invalid input: event must be an object");
  }

  const { customer, items, taxRate = 0.18, discountCode } = event;

  if (!customer || typeof customer !== "string") {
    throw new Error("customer name is required");
  }

  if (!Array.isArray(items) || items.length === 0) {
    throw new Error("items must be a non-empty array");
  }

  let subtotal = 0;

  for (const item of items) {
    if (
      !item.name ||
      typeof item.price !== "number" ||
      typeof item.quantity !== "number"
    ) {
      throw new Error("each item must have name, price, quantity");
    }

    subtotal += item.price * item.quantity;
  }

  let discount = 0;

  // simulate discount logic
  if (discountCode === "SAVE10") {
    discount = subtotal * 0.1;
  } else if (discountCode === "SAVE20") {
    discount = subtotal * 0.2;
  }

  const taxedAmount = (subtotal - discount) * taxRate;
  const total = subtotal - discount + taxedAmount;

  // simulate async DB/log call
  await new Promise((resolve) => setTimeout(resolve, 200));

  return {
    customer,
    itemsCount: items.length,
    subtotal,
    discount,
    tax: taxedAmount,
    total,
    currency: "INR",
    processedAt: new Date().toISOString(),
  };
};