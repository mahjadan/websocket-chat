//Saved nickname from previous usage
let nickname = window.localStorage.getItem("websocket-chat-username");
console.log("loading script.js")

//Nickname button and input
let nickI = document.getElementById("user_text");
let nickB = document.getElementById("user_btn");

//Chatting button and input
let chatTxt = document.getElementById("chat_text");
let chatBtn = document.getElementById("chat_btn");

// leave button
let leaveBtn = document.getElementById("leave-btn");

let join_modal = document.getElementById("join-modal");

//Gets websocket url by replacing http with ws and https with wss, and appending /websocket to the host
let protocol = location.protocol == "http:" ? "ws:" : "wss:";
let url = protocol + "//" + location.host + "/ws";

if (nickname != null) {
	nickI.value = nickname;
}

let ws = new WebSocket(url);

// on window close close the ws connection, remove localStorage
window.onbeforeunload = function (){
	ws.close(1000,"leaving , window closed")
	window.localStorage.removeItem("websocket-chat-username");
	return true
}


ws.onopen = function() {
	join_modal.style.display = "block";
	nickI.focus();
}

ws.onclose = function(e) {
	console.log("connection closed")
	console.log(e)

}
// todo how to hande onClose (to send leave event)
ws.onmessage = function(info) {
	let data = JSON.parse(info.data);
	console.log("new-message")
	handleMessage(data, info);
}

// todo handle error if user already exists, should return msg so front can show error message
function handleMessage(data, info) {
	switch (data.MessageType) {
		case "ONLINE_USERS":
			console.log('inside ONLINE_USERS')

			// rebuilding the online user list.
			// (race condition here), because when u rebuild the map, someone might leave the room, and try to access the map to delete it
			// check userLeft() method


			setOnlineUsers(data.Content)
			break;

		case "JOINED":  // this event only comes once, when the user is successfully connected
			console.log("joined Succesfully , asking for ONLINE_users")
			window.localStorage.setItem("websocket-chat-username", nickname);
			join_modal.style.display = "none";
			setProfile(data.Username)
			chatTxt.focus()
			// ask for online user list
			sendMessage("","ONLINE_USERS")
			break;

		case "CHAT":
			chatMessage(data.Username, data.Content, data.Date)
			break;

		case "ALREADY_EXISTS":
			toast(data.Content)
			break;

		case "SOMEONE_LEFT":
			userLeft(data.Username, data.Date);
			break;

		case "SOMEONE_JOIN":
			userJoin(data.Content,new Date())
			break;

		default:
			console.log("invalid type")
			console.log(info.data)
	}
}
/*
 * Leaving Chat room
 */

leaveBtn.onclick = function() {
	// sendMessage({ Username: nickname ,MessageType : "LEAVE", Date: new Date()})
	ws.close(1000,"leaving ")
	location.reload()
}

function chatMessage(sender, content, date) {
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

	document.getElementById("user-list").appendChild(p);
	// write it on the message box
	serverMessage(user + " has joined the server", date);
}

function setOnlineUsers(user_list) {
	// document.getElementById("user-list").innerHTML = ""; //remove old user_list
	// console.log("running the looooooooop")
	// console.log(online_users_list.size)
	// online_users_list.forEach((value, key) => {
	for (let i = 0; i < user_list.length; i++) {


		if (user_list[i] === nickname) {
			return
		}
		let p = document.createElement("p");
		p.className = user_list[i];
		p.innerText = user_list[i];
		console.log(p)
		document.getElementById("user-list").appendChild(p);
	}
	// })
}

function userLeft(user, date) {
	let userList = document.getElementById("user-list");
	userList.removeChild(userList.getElementsByClassName(user)[0]);
	serverMessage(user + " has left the server", date);
}

function setProfile(user) {
	let p = document.createElement("div");
	p.className = 'profile';
	p.innerText = user;

	document.getElementById("profile").appendChild(p);
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

	sendMessage("", "JOIN")
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

	sendMessage( chatTxt.value, "CHAT")
	chatTxt.value = "";
	chatTxt.focus()
}
chatBtn.onclick = function() {
	if(chatTxt.value.trim().length == 0) return;

	sendMessage(chatTxt.value,"CHAT")
	chatTxt.value = "";
	chatTxt.focus()
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
	return ` ${day} ${month} ${year} at ${hour}:${minute}`;
}

function createElm(name, classes) {
	let x = document.createElement(name);
	x.className = classes;
	return x;
}

function sendMessage(content,msgType){
	let msg = {Content: content, Username: nickname ,MessageType: msgType, Date: new Date()}
	ws.send(JSON.stringify(msg))
}