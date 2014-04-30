var connection      //websocket connection

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

function NewUser() {
    userName = document.getElementById("userNameTextBox").value
    if ('WebSocket' in window){
        /* WebSocket is supported. You can proceed with your code*/
        connection = new WebSocket("ws://localhost:1050/join?username="+userName)

        connection.onopen = function(){
           /*Send a small message to the console once the connection is established */
           alert('Connection open!');

           document.getElementById("startPage").style.display = "none";

           document.getElementById("mainPage").style.display = "block";

        }

        /*
            To send messages to this function from server do something like this:
                var message = {
                'name': 'bill murray',
                'comment': 'No one will ever believe you'
                };
            connection.send(JSON.stringify(message));
        */

        connection.onmessage = function(e){
           var server_message = e.data;
           console.log(server_message)

           var msg = JSON.parse(server_message);
           var user = msg.User
           var content = msg.Content
           var time = msg.TimeString

           var messageHolder = document.getElementById("messageHolder");
           var numRows = messageHolder.getElementsByTagName('tr').length;

           var row = messageHolder.insertRow(numRows);
           var cell1 = row.insertCell(0);
           var cell2 = row.insertCell(1);
           var cell3 = row.insertCell(2);

           cell1.innerHTML = user +" : "
           cell2.innerHTML = content;
           cell3.innerHTML = time;
           cell1.style.width = '100px';
           cell2.style.width = '500px';
           cell3.style.width = '100px';
        }

    } else {
        /*WebSockets are not supported. Try a fallback method like long-polling etc*/
        alert("Sorry you can't use our awesome chatterbox because your browser doesn't support websockets")
    }



}  

function SendMessage(msg) {
    msg = document.getElementById("message").value;
	connection.send(msg);
	document.getElementById("message").value = ""
}

function AddRoom(roomName){

}

