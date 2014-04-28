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
    if ('WebSocket' in window){
        /* WebSocket is supported. You can proceed with your code*/
        var connection = new WebSocket('ws://localhost:1050');

        connection.onopen = function(){
           /*Send a small message to the console once the connection is established */
           alert('Connection open!');
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
           console.log(server_message);
        }

    } else {
        /*WebSockets are not supported. Try a fallback method like long-polling etc*/
        alert("Sorry you can't use our awesome chatterbox because your browser doesn't support websockets")
    }
}  

function SendMessage(msg) {
	
}

function AddRoom(roomName){

}
