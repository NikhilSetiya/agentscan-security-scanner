// Vulnerable JavaScript code samples for AgentScan demo
// These examples contain intentional security vulnerabilities for demonstration purposes

const express = require('express');
const app = express();

// XSS Vulnerability - Reflected XSS
app.get('/search', (req, res) => {
    const query = req.query.q;
    // VULNERABLE: Direct output without sanitization
    res.send(`<h1>Search Results for: ${query}</h1>`);
});

// XSS Vulnerability - Stored XSS
app.post('/comment', (req, res) => {
    const comment = req.body.comment;
    // VULNERABLE: Storing and displaying user input without sanitization
    comments.push(comment);
    res.redirect('/comments');
});

app.get('/comments', (req, res) => {
    let html = '<h1>Comments</h1>';
    comments.forEach(comment => {
        // VULNERABLE: Direct output of stored user content
        html += `<div>${comment}</div>`;
    });
    res.send(html);
});

// DOM-based XSS
app.get('/profile', (req, res) => {
    res.send(`
        <script>
            const username = new URLSearchParams(window.location.search).get('name');
            // VULNERABLE: Direct DOM manipulation with user input
            document.getElementById('welcome').innerHTML = 'Welcome ' + username;
        </script>
        <div id="welcome"></div>
    `);
});

// SQL Injection Vulnerability
const mysql = require('mysql');
const connection = mysql.createConnection({
    host: 'localhost',
    user: 'root',
    password: 'password',
    database: 'myapp'
});

app.get('/user/:id', (req, res) => {
    const userId = req.params.id;
    // VULNERABLE: Direct string concatenation in SQL query
    const query = `SELECT * FROM users WHERE id = ${userId}`;
    
    connection.query(query, (error, results) => {
        if (error) throw error;
        res.json(results);
    });
});

// Command Injection Vulnerability
const { exec } = require('child_process');

app.get('/ping', (req, res) => {
    const host = req.query.host;
    // VULNERABLE: Direct command execution with user input
    exec(`ping -c 1 ${host}`, (error, stdout, stderr) => {
        if (error) {
            res.status(500).send('Error executing ping');
            return;
        }
        res.send(`<pre>${stdout}</pre>`);
    });
});

// Path Traversal Vulnerability
const fs = require('fs');
const path = require('path');

app.get('/file', (req, res) => {
    const filename = req.query.name;
    // VULNERABLE: Direct file access without path validation
    const filePath = path.join(__dirname, 'uploads', filename);
    
    fs.readFile(filePath, 'utf8', (err, data) => {
        if (err) {
            res.status(404).send('File not found');
            return;
        }
        res.send(data);
    });
});

// Insecure Cryptography
const crypto = require('crypto');

function hashPassword(password) {
    // VULNERABLE: Using MD5 for password hashing
    return crypto.createHash('md5').update(password).digest('hex');
}

function encryptData(data) {
    // VULNERABLE: Using deprecated DES encryption
    const cipher = crypto.createCipher('des', 'weak-key');
    let encrypted = cipher.update(data, 'utf8', 'hex');
    encrypted += cipher.final('hex');
    return encrypted;
}

// Hardcoded Secrets
const API_KEY = 'sk-1234567890abcdef'; // VULNERABLE: Hardcoded API key
const DB_PASSWORD = 'admin123'; // VULNERABLE: Hardcoded database password
const JWT_SECRET = 'my-secret-key'; // VULNERABLE: Hardcoded JWT secret

// Insecure Random Number Generation
function generateToken() {
    // VULNERABLE: Using Math.random() for security-sensitive operations
    return Math.random().toString(36).substring(2, 15);
}

// Prototype Pollution
app.post('/merge', (req, res) => {
    const userInput = req.body;
    const target = {};
    
    // VULNERABLE: Unsafe object merging
    function merge(target, source) {
        for (let key in source) {
            if (typeof source[key] === 'object' && source[key] !== null) {
                if (!target[key]) target[key] = {};
                merge(target[key], source[key]);
            } else {
                target[key] = source[key];
            }
        }
    }
    
    merge(target, userInput);
    res.json(target);
});

// Insecure Deserialization
app.post('/deserialize', (req, res) => {
    const serializedData = req.body.data;
    
    try {
        // VULNERABLE: Deserializing untrusted data
        const obj = eval('(' + serializedData + ')');
        res.json(obj);
    } catch (error) {
        res.status(400).send('Invalid data');
    }
});

// LDAP Injection
const ldap = require('ldapjs');

app.post('/login', (req, res) => {
    const username = req.body.username;
    const password = req.body.password;
    
    // VULNERABLE: LDAP injection through string concatenation
    const filter = `(&(uid=${username})(password=${password}))`;
    
    const client = ldap.createClient({
        url: 'ldap://localhost:389'
    });
    
    client.search('ou=users,dc=example,dc=com', {
        filter: filter,
        scope: 'sub'
    }, (err, search) => {
        // Handle search results
    });
});

// XML External Entity (XXE) Vulnerability
const xml2js = require('xml2js');

app.post('/xml', (req, res) => {
    const xmlData = req.body.xml;
    
    // VULNERABLE: XML parsing without disabling external entities
    const parser = new xml2js.Parser();
    
    parser.parseString(xmlData, (err, result) => {
        if (err) {
            res.status(400).send('Invalid XML');
            return;
        }
        res.json(result);
    });
});

module.exports = app;