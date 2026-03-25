#!/usr/bin/env python3
"""
HTTP API Tests for Account Service
Tests all endpoints: register, login, refresh, profile, revoke
"""

import requests
import json
import time
import subprocess
import signal
import os
import sys
from typing import Dict, Optional

class AccountServiceTester:
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.server_process = None
        self.access_token = None
        self.refresh_token = None

    def check_server(self):
        """Check if server is running"""
        print("🔍 Checking if server is running...")

        try:
            response = requests.get(f"{self.base_url}/health", timeout=5)
            if response.status_code == 200:
                print("✅ Server is running")
                return True
            else:
                print(f"❌ Server health check failed: {response.status_code}")
                return False
        except Exception as e:
            print(f"❌ Cannot connect to server: {e}")
            return False

    def test_register(self, username: str, password: str) -> bool:
        """Test user registration"""
        print(f"📝 Testing registration for user: {username}")

        url = f"{self.base_url}/register-local"
        data = {"username": username, "password": password}

        try:
            response = requests.post(url, json=data, timeout=5)
            print(f"   Status: {response.status_code}")
            print(f"   Request: POST {url}")
            print(f"   Body: {json.dumps(data, indent=2)}")

            if response.status_code == 201:
                result = response.json()
                print(f"   Response: {json.dumps(result, indent=2)}")
                self.access_token = result.get("access_token")
                self.refresh_token = result.get("refresh_token")
                print("   ✅ Registration successful")
                return True
            else:
                try:
                    error_result = response.json()
                    print(f"   Error Response: {json.dumps(error_result, indent=2)}")
                except:
                    print(f"   Error Response: {response.text}")
                print("   ❌ Registration failed")
                return False

        except Exception as e:
            print(f"   ❌ Request failed: {e}")
            return False

    def test_login(self, username: str, password: str) -> bool:
        """Test user login"""
        print(f"🔐 Testing login for user: {username}")

        url = f"{self.base_url}/login-local"
        data = {"username": username, "password": password}

        try:
            response = requests.post(url, json=data, timeout=5)
            print(f"   Status: {response.status_code}")
            print(f"   Request: POST {url}")
            print(f"   Body: {json.dumps(data, indent=2)}")

            if response.status_code == 200:
                result = response.json()
                print(f"   Response: {json.dumps(result, indent=2)}")
                self.access_token = result.get("access_token")
                self.refresh_token = result.get("refresh_token")
                print("   ✅ Login successful")
                return True
            else:
                try:
                    error_result = response.json()
                    print(f"   Error Response: {json.dumps(error_result, indent=2)}")
                except:
                    print(f"   Error Response: {response.text}")
                print("   ❌ Login failed")
                return False

        except Exception as e:
            print(f"   ❌ Request failed: {e}")
            return False

    def test_refresh_token(self) -> bool:
        """Test token refresh"""
        print("🔄 Testing token refresh")

        if not self.refresh_token:
            print("   ❌ No refresh token available")
            return False

        url = f"{self.base_url}/refresh-token"
        data = {"refresh_token": self.refresh_token}

        try:
            response = requests.post(url, json=data, timeout=5)
            print(f"   Status: {response.status_code}")

            if response.status_code == 200:
                result = response.json()
                new_access_token = result.get("access_token")
                print("   ✅ Token refresh successful")
                print(f"   New access token: {new_access_token[:20]}...")
                self.access_token = new_access_token
                return True
            else:
                print(f"   ❌ Token refresh failed: {response.text}")
                return False

        except Exception as e:
            print(f"   ❌ Request failed: {e}")
            return False

    def test_profile(self) -> bool:
        """Test profile access"""
        print("👤 Testing profile access")

        if not self.access_token:
            print("   ❌ No access token available")
            return False

        url = f"{self.base_url}/profile"
        headers = {"Authorization": f"Bearer {self.access_token}"}

        try:
            response = requests.get(url, headers=headers, timeout=5)
            print(f"   Status: {response.status_code}")

            if response.status_code == 200:
                profile = response.json()
                print("   ✅ Profile access successful")
                print(f"   Username: {profile.get('App', {}).get('Username', 'N/A')}")
                return True
            else:
                print(f"   ❌ Profile access failed: {response.text}")
                return False

        except Exception as e:
            print(f"   ❌ Request failed: {e}")
            return False

    def test_revoke_token(self) -> bool:
        """Test token revocation"""
        print("🚫 Testing token revocation")

        if not self.refresh_token:
            print("   ❌ No refresh token available")
            return False

        url = f"{self.base_url}/revoke-token"
        data = {"refresh_token": self.refresh_token}

        try:
            response = requests.delete(url, json=data, timeout=5)
            print(f"   Status: {response.status_code}")

            if response.status_code == 200:
                result = response.json()
                print("   ✅ Token revocation successful")
                print(f"   Message: {result.get('message', 'N/A')}")
                return True
            else:
                print(f"   ❌ Token revocation failed: {response.text}")
                return False

        except Exception as e:
            print(f"   ❌ Request failed: {e}")
            return False

    def test_invalid_login(self) -> bool:
        """Test invalid login (should fail)"""
        print("❌ Testing invalid login")

        url = f"{self.base_url}/login-local"
        data = {"username": "nonexistent", "password": "wrongpass"}

        try:
            response = requests.post(url, json=data, timeout=5)
            print(f"   Status: {response.status_code}")

            if response.status_code == 401:
                print("   ✅ Invalid login correctly rejected")
                return True
            else:
                print(f"   ❌ Invalid login not rejected: {response.text}")
                return False

        except Exception as e:
            print(f"   ❌ Request failed: {e}")
            return False

    def check_logs(self):
        """Check if logs are being written"""
        print("📄 Checking logs...")

        log_file = "logs/app.log"
        if os.path.exists(log_file):
            with open(log_file, 'r') as f:
                lines = f.readlines()
                if lines:
                    print(f"   ✅ Found {len(lines)} log entries")
                    # Show last 3 entries
                    for line in lines[-3:]:
                        try:
                            log_entry = json.loads(line.strip())
                            level = log_entry.get('level', 'UNKNOWN')
                            msg = log_entry.get('msg', 'No message')
                            print(f"   📝 {level}: {msg}")
                        except:
                            print(f"   📝 {line.strip()[:100]}...")
                else:
                    print("   ⚠️  Log file exists but is empty")
        else:
            print("   ❌ Log file not found")

    def print_api_spec(self):
        """Print API specification"""
        print("📋 Account Service API Specification")
        print("=" * 60)

        api_endpoints = [
            {
                "method": "GET",
                "path": "/health",
                "description": "Health check",
                "auth": "None",
                "body": "None",
                "response": '{"status": "ok"}'
            },
            {
                "method": "POST",
                "path": "/register-local",
                "description": "User registration",
                "auth": "None",
                "body": '{"username": "string", "password": "string"}',
                "response": '{"access_token": "...", "refresh_token": "..."}'
            },
            {
                "method": "POST",
                "path": "/login-local",
                "description": "User login",
                "auth": "None",
                "body": '{"username": "string", "password": "string"}',
                "response": '{"access_token": "...", "refresh_token": "..."}'
            },
            {
                "method": "GET",
                "path": "/profile",
                "description": "Get user profile",
                "auth": "Bearer Token",
                "body": "None",
                "response": '{"App": {"Username": "...", ...}}'
            },
            {
                "method": "POST",
                "path": "/refresh-token",
                "description": "Refresh access token",
                "auth": "None",
                "body": '{"refresh_token": "string"}',
                "response": '{"access_token": "..."}'
            },
            {
                "method": "DELETE",
                "path": "/revoke-token",
                "description": "Revoke refresh token",
                "auth": "None",
                "body": '{"refresh_token": "string"}',
                "response": '{"message": "true"}'
            }
        ]

        for endpoint in api_endpoints:
            print(f"\n🔗 {endpoint['method']} {endpoint['path']}")
            print(f"   📝 {endpoint['description']}")
            print(f"   🔐 Auth: {endpoint['auth']}")
            print(f"   📦 Body: {endpoint['body']}")
            print(f"   📤 Response: {endpoint['response']}")

        print("\n" + "=" * 60)

    def run_all_tests(self):
        """Run all tests"""
        print("🧪 Starting HTTP API tests for Account Service")
        print("=" * 50)

        # Print API specification
        self.print_api_spec()

        # Check server
        if not self.check_server():
            print("❌ Server is not running. Please start it first.")
            return False

        # Test data
        test_username = f"testuser_{int(time.time())}"
        test_password = "testpass123"

        # Run tests
        tests = [
            ("Register", lambda: self.test_register(test_username, test_password)),
            ("Login", lambda: self.test_login(test_username, test_password)),
            ("Profile", self.test_profile),
            ("Refresh Token", self.test_refresh_token),
            ("Invalid Login", self.test_invalid_login),
            ("Revoke Token", self.test_revoke_token),
        ]

        results = []
        for test_name, test_func in tests:
            print(f"\n🔬 Running test: {test_name}")
            result = test_func()
            results.append((test_name, result))

        # Summary
        print("\n" + "=" * 50)
        print("📊 Test Results:")

        passed = 0
        total = len(results)

        for test_name, result in results:
            status = "✅ PASS" if result else "❌ FAIL"
            print(f"   {test_name}: {status}")
            if result:
                passed += 1

        print(f"\n🎯 Summary: {passed}/{total} tests passed")

        # Check logs
        print("\n" + "=" * 50)
        self.check_logs()

        return passed == total


if __name__ == "__main__":
    tester = AccountServiceTester()

    # Change back to project root
    os.chdir("..")

    success = tester.run_all_tests()

    if success:
        print("\n🎉 All tests passed!")
        sys.exit(0)
    else:
        print("\n💥 Some tests failed!")
        sys.exit(1)