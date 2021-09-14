# REST API

## Get Tree

### GET `http://localhost:{port}/tree`

### Arguments
- link (string): the root URL of the tree
- depth (int): the depth of the tree

e.g. depth of 1
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

### GET `http://localhost:{port}/emails`

### Arguments
- link (string): the root URL of the tree

```json
["random@gmail.com", "random@yahoo.com"]
```

## Get IP

### GET `http://localhost:{port}/ip`

```
"Random IP Address"
```


