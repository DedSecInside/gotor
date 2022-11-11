# REST HTTP API (Examples are using localhost)

## Get Link Tree
### GET `http://localhost:{port}/tree?link=https://example.com&depth=1`
### Arguments
- link (string): the root URL of the tree
- depth (int): the depth of the tree
### Response
```json
{
        "url": "https://www.example.com",
        "status": "OK",
        "status_code": 200,
        "children": [{
                "url": "https://www.child.com",
                "status": "OK",
                "status_code": 200,
                "children": []
        }]
}
```

## Get Emails
### GET `http://localhost:{port}/emails?link=https://random.com`
### Arguments
- link (string): the root URL of the tree
### Response
```json
["random@gmail.com", "random@yahoo.com"]
```

## Get Phone Numbers
### GET `http://localhost:{port}/phone_numbers?link=https://example.com` 
### Arguments
- link (string): the root URL of the tree
### Response
```json
["+1-234-567-8901", "+1-234-567-8902"]
```

## Get current IP of server
### GET `http://localhost:{port}/ip`
### Arguments
N/A
### Response
```json
"127.0.0.1" (returns IP address as plain string)
```


