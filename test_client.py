#!/usr/bin/env python3
"""Test client for macos-notify-bridge"""

import socket
import json
import sys

def send_notification(title, message, host='localhost', port=9876):
    """Send a notification to the macos-notify-bridge server"""
    try:
        # Create socket connection
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.connect((host, port))
            
            # Prepare and send data
            data = json.dumps({'title': title, 'message': message}) + '\n'
            s.sendall(data.encode())
            
            # Receive response
            response = s.recv(1024).decode().strip()
            return response
    except ConnectionRefusedError:
        return "ERROR: Connection refused. Is the server running?"
    except Exception as e:
        return f"ERROR: {str(e)}"

if __name__ == "__main__":
    print("Testing macos-notify-bridge...")
    
    # Test 1: Valid notification
    print("\n1. Sending valid notification:")
    result = send_notification("Test Notification", "Hello from Python test client!")
    print(f"   Response: {result}")
    
    # Test 2: Another valid notification
    print("\n2. Sending another notification:")
    result = send_notification("Build Status", "Build completed successfully!")
    print(f"   Response: {result}")
    
    # Test 3: Invalid JSON test (if running server separately)
    print("\n3. Testing error handling:")
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.connect(('localhost', 9876))
            s.sendall(b'invalid json\n')
            response = s.recv(1024).decode().strip()
            print(f"   Response: {response}")
    except Exception as e:
        print(f"   Error: {str(e)}")
    
    print("\nTest completed!")