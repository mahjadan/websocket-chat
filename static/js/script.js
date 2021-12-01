//Saved nickname from previous usage
let nickname = window.localStorage.getItem("websocket-chat-username");
console.log("loading script.js")

//Nickname button and input
let nickI = document.getElementById("user_text");
let nickB = document.getElementById("user_btn");

//Chatting button and input
let chatTxt = document.getElementById("chat_text");
let chatBtn = document.getElementById("chat_btn");

let registered = false;
let modal = document.getElementById("modal");

//Gets websocket url by replacing http with ws and https with wss, and appending /websocket to the host
let protocol = location.protocol == "http:" ? "ws:" : "wss:";
let url = protocol + "//" + location.host + "/ws";

if (nickname != null) {
	nickI.value = nickname;
}

let ws = new WebSocket(url);

ws.onopen = function() {
	modal.style.display = "block";
	nickI.focus();
}
// todo handle error if user already exists, should return msg so front can show error message
function handleMessage(data, info) {
	switch (data.MessageType) {
		case "USER_LIST":
			console.log('inside userlist')
			console.log(data.Content)
			document.getElementById("users").innerHTML = ""; //remove old user_list

			for (let i = 0; i < data.Content.length; i++) { // add the new list
				userJoin(data.Content[i]);
			}
			break;
		case "JOINED":
			console.log("insided joined")
			registered = true;
			window.localStorage.setItem("websocket-chat-username", nickname);
			modal.style.display = "none";
			chatTxt.focus()
			break;

		case "CHAT":
			chatMessage(data.Username, data.Content, data.Date)
			break;

		case "ALREADY_EXISTS":
			toast(data.Content)
			break;

		default:
			console.log("invalid type")
			console.log(info.data)
	}
}

// todo how to hande onClose (to send leave event)
ws.onmessage = function(info) {
	let data = JSON.parse(info.data);
	console.log("new-message")
	handleMessage(data, info);
}

/*
 * Message receive handling
 */

function newMessage(data) {
	switch(data.type) {
		case "join":
			userJoin(data.content, data.date);
			serverMessage(data.content + " has joined the server", data.date);
			break;
		case "leave":
			userLeave(data.content, data.date);
			break;
		case "message":
			chatMessage(data.sender, data.content, data.date)
	}
}

function chatMessage(sender, content, date) {
	console.log(date)
	let msg = createElm("div", "message");

	aClasses = "author"
	if (sender === nickname) {
		aClasses += " self"
	}
	let author = createElm("div", aClasses);
	author.innerText = sender;

	let timestamp = createElm("div", "timestamp");
	timestamp.innerText = dateToString(new Date(date));

	let text = createElm("div", "content");
	text.innerText = content;

	msg.append(author, timestamp, text);
	insertMessage(msg);
}

function serverMessage(content, date) {
	let msg = createElm("div", "message");

	let text = createElm("div", "server-message");
	text.innerText = content;

	msg.append(text);
	insertMessage(msg);
}

function insertMessage(div) {
	//If message scrolling at the bottom, move to bottom again to update
	let scroller = document.getElementsByClassName("messages")[0];
	let currentScroll = scroller.scrollTop;
	let maxScroll = scroller.scrollHeight - scroller.clientHeight;

	document.getElementById("messages").append(div);

	if (currentScroll === maxScroll) {
		div.scrollIntoView();
	}
}
/**
 * Join/leave events
 */
function userJoin(user, date) {
	let p = document.createElement("p");
	p.className = user;
	p.innerText = user;

	document.getElementById("users").appendChild(p);
}

function userLeave(user, date) {
	users = document.getElementById("users");
	users.removeChild(users.getElementsByClassName(user)[0]);
	serverMessage(user + " has left the server", date);
}

/*
 * Registration of Username
 */
function register() {
	console.log("trying to register...")
	//todo validate username
	let nick = nickI.value;
	if (nick.trim().length == 0) return;
	nickname = nick;

	sendMessage({Date: new Date(), Username: nick, MessageType : "JOIN"})
}

nickI.onkeydown = function(ev) {
	if(ev.key !== "Enter") {
		return;
	}
	register();
}

nickB.onclick = function() {
	console.log("clicked")
	register();
}

/**
 * Message sending
 */

chatTxt.onkeydown = function(ev) {
	if(ev.key !== "Enter") {
		return;
	}
	if(chatTxt.value.trim().length == 0) return;

	// ws.send(chatTxt.value);
	sendMessage({Content: chatTxt.value, Username: nickname ,MessageType: "CHAT"})
	chatTxt.value = "";
}
chatBtn.onclick = function() {
	if(chatTxt.value.trim().length == 0) return;

	sendMessage({Content: chatTxt.value, Username: nickname ,MessageType: "CHAT"})
	chatTxt.value = "";
}

/*
 * Util functions
 */
function toast(message) {
	console.log('calling toast')
	var x = document.getElementById("snackbar");
	x.innerText = message;
	x.className = "snackshow";
	setTimeout(function(){ x.className = x.className.replace("snackshow", ""); }, 3000);
}

let months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
function dateToString(date) {
	let year = date.getFullYear();
	let month = months[date.getMonth()];
	let day = date.getDate();
	let hour = date.getHours();
	let minute = date.getMinutes();
	console.log(date)
	console.log(` ${day} ${month} ${year} at ${hour}:${minute}`)
	return ` ${day} ${month} ${year} at ${hour}:${minute}`;
}

function createElm(name, classes) {
	let x = document.createElement(name);
	x.className = classes;
	return x;
}

function sendMessage(msg){
	ws.send(JSON.stringify(msg))
}