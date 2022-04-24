let fingerprintDOM = document.getElementById("fingerprint");
let messageDOM = document.getElementById("message");
let sendDOM = document.getElementById("send");
let chatboxDOM = document.getElementById("chatbox");

let key;

let requests = {};

let messages = new EventTarget();

function generateID(length) {
    let result = '';
    let characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let charactersLength = characters.length;
    for (let i = 0; i < length; i++) {
        result += characters.charAt(Math.floor(Math.random() *
            charactersLength));
    }
    return result;
}

messages.addEventListener("message", async function (msg) {
    let message = msg.detail;
    let key = await importKey(fromHex(message.key));
    if (!await verifySignature(key, fromHex(message.signature), message.message)) {
        console.log("Message verification failed");
        return;
    }

    let container = document.createElement("div");
    let fingerprint = document.createElement("span");
    fingerprint.classList.add("fingerprint");
    fingerprint.innerText = await getFingerprint(fromHex(message.key));
    container.appendChild(fingerprint);
    let text = document.createElement("span");
    text.innerText = message.message;
    container.appendChild(text);
    chatboxDOM.appendChild(container);

    chatboxDOM.scrollTop = chatboxDOM.scrollHeight;
});

function createRequest(type, request) {
    let id = generateID(8);

    return {
        id: id,
        type: type,
        request: request
    }
}

function sendRequest(ws, request) {
    let promise = new Promise((resolve, reject) => {
        requests[request.id] = {
            resolve: resolve,
            reject: reject
        }
    });

    ws.send(JSON.stringify(request));
    return promise;
}

function resolveRequest(response) {
    if (response.success) {
        requests[response.id].resolve(response.result);
    } else {
        requests[response.id].reject(response.error);
    }

    delete requests[response.id];
}

function updateFingerprint(newFingerprint) {
    fingerprintDOM.innerText = `Your key fingerprint: ${newFingerprint}`;
}

async function sendMessage() {
    if (!messageDOM.value) {
        return;
    }
    let signature = await signMessage(key, messageDOM.value);
    let repr = await getKeyRepresentation(key);
    await sendRequest(ws, createRequest("publish", {
        key: repr,
        message: messageDOM.value,
        signature: toHex(signature).toUpperCase()
    }));
    messageDOM.value = "";
}

messageDOM.addEventListener("keypress", async function (ev) {
    if (ev.code === "Enter") {
        await sendMessage();
    }
})

sendDOM.addEventListener("click", async function () {
    await sendMessage();
});

let ws;

(async function () {
    key = await generateKeyPair();
    updateFingerprint(await getKeyPairFingerprint(key));
    console.log(toHex(await signMessage(key, "hello world")).toUpperCase());

    ws = new WebSocket("ws://localhost:8080/ws");

    ws.addEventListener("message", function (message) {
        let req = JSON.parse(message.data);
        if (req.type !== "response") {
            messages.dispatchEvent(new CustomEvent("message", {
                detail: req.result
            }));
            return;
        }
        resolveRequest(req);
    })
})();