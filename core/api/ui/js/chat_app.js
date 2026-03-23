import { loadNavBar } from "./components/navbar.js";
import { A2UIRenderer } from "./a2ui/renderer.js";

// Load standard Navbar
const navBox = document.querySelector('#nav-box');
loadNavBar(navBox);

// Initialize A2UI Renderer
const surface = document.querySelector('#surface-root');
const renderer = new A2UIRenderer(
    surface, 
    '/api/a2ui/stream', 
    '/api/a2ui/action'
);
renderer.start();

// Setup Chat Logic
const inputField = document.getElementById('chat-input');
const sendButton = document.getElementById('chat-send');
const historyBox = document.getElementById('chat-history');

function appendMessage(text, role) {
    const msgDiv = document.createElement('div');
    msgDiv.className = `message ${role}`;
    msgDiv.textContent = text;
    historyBox.appendChild(msgDiv);
    historyBox.scrollTop = historyBox.scrollHeight;
}

async function sendMessage() {
    const text = inputField.value.trim();
    if (!text) return;

    inputField.value = '';
    appendMessage(text, 'user');

    try {
        const response = await fetch('/api/chat/message', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ message: text })
        });

        const data = await response.json();
        
        if (data.error) {
            appendMessage(`Error: ${data.error}`, 'agent');
        } else {
            appendMessage(data.response, 'agent');
        }
    } catch (err) {
        appendMessage(`Failed to send message: ${err.message}`, 'agent');
    }
}

sendButton.addEventListener('click', sendMessage);
inputField.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        sendMessage();
    }
});
