"""Security-focused tests for image URL validation."""

import os
import unittest
from unittest.mock import patch

from app.utils.image_validator import validate_file_signature, validate_url


class ImageValidatorTests(unittest.TestCase):
    def test_remote_urls_are_disabled_by_default(self) -> None:
        with patch.dict(os.environ, {}, clear=True):
            self.assertFalse(validate_url("https://example.com/beetle.jpg"))

    def test_allow_list_restricts_enabled_remote_urls(self) -> None:
        environment = {
            "ENABLE_REMOTE_IMAGE_URLS": "true",
            "ALLOWED_IMAGE_HOSTS": "images.example.com",
        }
        with patch.dict(os.environ, environment, clear=True):
            self.assertTrue(validate_url("https://images.example.com/beetle.jpg"))
            self.assertFalse(validate_url("https://example.com/beetle.jpg"))

    def test_rejects_credentials_in_url(self) -> None:
        with patch.dict(os.environ, {"ENABLE_REMOTE_IMAGE_URLS": "true"}, clear=True):
            self.assertFalse(validate_url("https://user:password@example.com/beetle.jpg"))

    def test_recognizes_supported_magic_number(self) -> None:
        self.assertTrue(validate_file_signature(b"\x89PNG\r\n\x1a\ncontent"))
        self.assertFalse(validate_file_signature(b"plain text"))


if __name__ == "__main__":
    unittest.main()
