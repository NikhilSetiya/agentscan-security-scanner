#!/usr/bin/env python3
"""
Vulnerable Python code samples for AgentScan demo
These examples contain intentional security vulnerabilities for demonstration purposes
"""

import sqlite3
import subprocess
import pickle
import os
import hashlib
import random
import xml.etree.ElementTree as ET
from flask import Flask, request, render_template_string
import yaml

app = Flask(__name__)

# SQL Injection Vulnerabilities
class UserDatabase:
    def __init__(self):
        self.conn = sqlite3.connect('users.db', check_same_thread=False)
        
    def get_user_by_id(self, user_id):
        # VULNERABLE: Direct string formatting in SQL query
        query = f"SELECT * FROM users WHERE id = {user_id}"
        cursor = self.conn.cursor()
        cursor.execute(query)
        return cursor.fetchone()
    
    def authenticate_user(self, username, password):
        # VULNERABLE: String concatenation in SQL query
        query = "SELECT * FROM users WHERE username = '" + username + "' AND password = '" + password + "'"
        cursor = self.conn.cursor()
        cursor.execute(query)
        return cursor.fetchone()
    
    def search_users(self, search_term):
        # VULNERABLE: % formatting in SQL query
        query = "SELECT * FROM users WHERE name LIKE '%%%s%%'" % search_term
        cursor = self.conn.cursor()
        cursor.execute(query)
        return cursor.fetchall()

# Command Injection Vulnerabilities
@app.route('/ping')
def ping_host():
    host = request.args.get('host', 'localhost')
    # VULNERABLE: Direct command execution with user input
    result = subprocess.run(f'ping -c 1 {host}', shell=True, capture_output=True, text=True)
    return f'<pre>{result.stdout}</pre>'

@app.route('/backup')
def backup_file():
    filename = request.args.get('file')
    # VULNERABLE: Command injection through os.system
    os.system(f'cp {filename} /backup/')
    return 'Backup completed'

def execute_command(cmd):
    # VULNERABLE: Using eval with user input
    return eval(f'subprocess.run("{cmd}", shell=True)')

# Path Traversal Vulnerabilities
@app.route('/download')
def download_file():
    filename = request.args.get('file')
    # VULNERABLE: Direct file access without path validation
    try:
        with open(f'/uploads/{filename}', 'r') as f:
            return f.read()
    except FileNotFoundError:
        return 'File not found', 404

def read_config_file(config_name):
    # VULNERABLE: Path traversal through string concatenation
    config_path = '/etc/myapp/' + config_name
    with open(config_path, 'r') as f:
        return f.read()

# Insecure Deserialization
@app.route('/deserialize', methods=['POST'])
def deserialize_data():
    data = request.get_data()
    # VULNERABLE: Deserializing untrusted data with pickle
    try:
        obj = pickle.loads(data)
        return str(obj)
    except Exception as e:
        return f'Error: {e}', 400

def load_user_session(session_data):
    # VULNERABLE: Using eval for deserialization
    return eval(session_data)

# Weak Cryptography
import hashlib

def hash_password(password):
    # VULNERABLE: Using MD5 for password hashing
    return hashlib.md5(password.encode()).hexdigest()

def generate_token():
    # VULNERABLE: Using weak random number generation
    return str(random.randint(1000, 9999))

# Hardcoded Secrets
API_KEY = 'sk-1234567890abcdef'  # VULNERABLE: Hardcoded API key
DATABASE_PASSWORD = 'admin123'   # VULNERABLE: Hardcoded database password
SECRET_KEY = 'my-flask-secret'   # VULNERABLE: Hardcoded secret key

# Server-Side Template Injection (SSTI)
@app.route('/profile')
def user_profile():
    username = request.args.get('name', 'Guest')
    # VULNERABLE: Direct template rendering with user input
    template = f'<h1>Welcome {username}!</h1>'
    return render_template_string(template)

@app.route('/render')
def render_template():
    template_content = request.args.get('template')
    # VULNERABLE: Rendering user-controlled template
    return render_template_string(template_content)

# XML External Entity (XXE) Vulnerability
@app.route('/xml', methods=['POST'])
def parse_xml():
    xml_data = request.get_data()
    # VULNERABLE: XML parsing without disabling external entities
    try:
        root = ET.fromstring(xml_data)
        return f'Parsed XML: {ET.tostring(root, encoding="unicode")}'
    except ET.ParseError as e:
        return f'XML Parse Error: {e}', 400

# YAML Deserialization Vulnerability
@app.route('/config', methods=['POST'])
def load_config():
    config_data = request.get_data(as_text=True)
    # VULNERABLE: Loading YAML without safe_load
    try:
        config = yaml.load(config_data, Loader=yaml.Loader)
        return str(config)
    except yaml.YAMLError as e:
        return f'YAML Error: {e}', 400

# LDAP Injection
import ldap

def authenticate_ldap(username, password):
    # VULNERABLE: LDAP injection through string formatting
    ldap_filter = f"(&(uid={username})(password={password}))"
    
    conn = ldap.initialize('ldap://localhost:389')
    try:
        result = conn.search_s('ou=users,dc=example,dc=com', ldap.SCOPE_SUBTREE, ldap_filter)
        return len(result) > 0
    except ldap.LDAPError:
        return False

# NoSQL Injection (MongoDB)
from pymongo import MongoClient

def find_user_mongo(username):
    client = MongoClient('mongodb://localhost:27017/')
    db = client.myapp
    
    # VULNERABLE: Direct user input in MongoDB query
    query = f'{{"username": "{username}"}}'
    # This would be vulnerable if eval is used to parse the query
    return db.users.find(eval(query))

# Race Condition Vulnerability
import threading
import time

balance = 1000
balance_lock = threading.Lock()

def withdraw_money(amount):
    global balance
    # VULNERABLE: Race condition - checking and updating balance without proper locking
    if balance >= amount:
        time.sleep(0.1)  # Simulate processing time
        balance -= amount
        return True
    return False

# Insecure Random Number Generation for Security
def generate_password_reset_token():
    # VULNERABLE: Using predictable random for security token
    return str(random.randint(100000, 999999))

def generate_session_id():
    # VULNERABLE: Weak randomness for session ID
    return hashlib.md5(str(random.random()).encode()).hexdigest()

# Information Disclosure
@app.route('/debug')
def debug_info():
    # VULNERABLE: Exposing sensitive debug information
    import sys
    debug_data = {
        'python_version': sys.version,
        'environment_variables': dict(os.environ),
        'current_directory': os.getcwd(),
        'process_id': os.getpid()
    }
    return debug_data

# Unsafe File Upload
@app.route('/upload', methods=['POST'])
def upload_file():
    if 'file' not in request.files:
        return 'No file uploaded', 400
    
    file = request.files['file']
    # VULNERABLE: No file type validation or size limits
    filename = file.filename
    file.save(f'/uploads/{filename}')
    return f'File {filename} uploaded successfully'

# Cross-Site Request Forgery (CSRF)
@app.route('/transfer', methods=['POST'])
def transfer_money():
    # VULNERABLE: No CSRF protection
    from_account = request.form.get('from')
    to_account = request.form.get('to')
    amount = request.form.get('amount')
    
    # Process transfer without CSRF token validation
    return f'Transferred ${amount} from {from_account} to {to_account}'

# Insecure Direct Object Reference
@app.route('/document/<doc_id>')
def get_document(doc_id):
    # VULNERABLE: No authorization check
    try:
        with open(f'/documents/{doc_id}.txt', 'r') as f:
            return f.read()
    except FileNotFoundError:
        return 'Document not found', 404

if __name__ == '__main__':
    # VULNERABLE: Running Flask in debug mode in production
    app.run(debug=True, host='0.0.0.0')