function JoinRoom() {
	roomName = document.getElementById("roomName").value
	
	var ul = document.getElementById("RoomPanel");
	
	var li = document.createElement("li"); 
	
	var a = document.createElement("a"); 
	a.href="#"+roomName;
	a.setAttribute("data-toggle", "tab");   
	a.innerHTML += roomName;
	
	li.appendChild(a);
	ul.appendChild(li);
}  

function NewUser(userName) {

}  

function SendMessage(msg) {
	
}

