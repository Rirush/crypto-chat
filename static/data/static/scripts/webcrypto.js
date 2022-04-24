const byteToHex = [];

for (let n = 0; n <= 0xff; ++n) {
    const hexOctet = n.toString(16).padStart(2, "0");
    byteToHex.push(hexOctet);
}

function toHex(arrayBuffer) {
    const buff = new Uint8Array(arrayBuffer);
    const hexOctets = []; // new Array(buff.length) is even faster (preallocates necessary array size), then use hexOctets[i] instead of .push()

    for (let i = 0; i < buff.length; ++i)
        hexOctets.push(byteToHex[buff[i]]);

    return hexOctets.join("");
}

function fromHex(hex) {
    return new Uint8Array(hex.match(/../g).map(function (h) {
        return parseInt(h, 16)
    }));
}

async function generateKeyPair() {
    return crypto.subtle.generateKey({
        name: "ECDSA",
        namedCurve: "P-384",
    }, true, ["sign", "verify"]);
}

async function getKeyPairFingerprint(key) {
    let rawPublicKey = await crypto.subtle.exportKey("raw", key.publicKey);
    let digest = await crypto.subtle.digest("SHA-256", rawPublicKey);
    return toHex(digest).toUpperCase();
}

async function getFingerprint(data) {
    let digest = await crypto.subtle.digest("SHA-256", data);
    return toHex(digest).toUpperCase();
}

async function getKeyRepresentation(key) {
    let rawPublicKey = await crypto.subtle.exportKey("raw", key.publicKey);
    return toHex(rawPublicKey).toUpperCase();
}

async function signMessage(key, message) {
    let encoder = new TextEncoder();
    let messageBytes = encoder.encode(message);

    return crypto.subtle.sign({name: "ECDSA", hash: "SHA-256"}, key.privateKey, messageBytes);
}

async function importKey(publicKey) {
    return crypto.subtle.importKey("raw", publicKey, {name: "ECDSA", namedCurve: "P-384"}, true, ["verify"]);
}

async function verifySignature(publicKey, signature, message) {
    let encoder = new TextEncoder();
    let messageBytes = encoder.encode(message);

    return crypto.subtle.verify({name: "ECDSA", hash: "SHA-256"}, publicKey, signature, messageBytes);
}