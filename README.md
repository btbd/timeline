# Timeline

A simple timeline for logging pictures with messages.

## Usage

### Server Arguments

- `p` - Port (default 80)
- `h` - Web page header title (default "Timeline")
- `v` - Display debug info
- `crt` - File path to certificate for TLS
- `key` - File path to key for TLS
- `tokens` - File path to Bearer tokens separated by new lines for verifying POST requests
	- If this is not specified, no Bearer token is necessary to add a post
	- Example tokens file
```
firstToken123 # comment
secondToken987 # another comment
```

### Adding Posts

- Make a multipart POST request to `host/post` with:
	- **HTTP Header `Authorization`** - Contains a valid Bearer token (if necessary)
	- **1st part** - JSON message containing `from` (author) and `message` (caption text for image)
	- **2nd part** - Raw image to post
	
#### Example POST using NodeJS
```js
var request = require("request");
var fs      = require("fs");

request({
	method: "POST",
	url: "http://localhost/post",
	headers: {
		"authorization": "Bearer token123",
		"content-type": "multipart/mixed"
	},
	multipart: [
		{
			"content-type": "application/json",
			body: JSON.stringify({
				"from": "the author",
				"message": "the message",
			})
		},
		{
			"content-type": "image/jpeg",
			body: fs.createReadStream("image.jpg")
		}
	],
}, function(error, response, body) {
	console.log(error, body);
});
```