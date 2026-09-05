"""Image format checks and guarded remote downloads."""

from __future__ import annotations

import asyncio
import ipaddress
import os
import socket
from urllib.parse import urljoin, urlparse

import httpx
import magic
from fastapi import HTTPException

MAX_FILE_SIZE = 10 * 1024 * 1024
MAX_REDIRECTS = 3
ALLOWED_MIME_TYPES = {"image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp"}
SIGNATURES = (
    b"\xff\xd8\xff",
    b"\x89PNG\r\n\x1a\n",
    b"GIF8",
    b"BM",
    b"RIFF",
)


def validate_url(url: str) -> bool:
    """Perform syntax and allow-list checks before resolving a remote URL."""

    if os.getenv("ENABLE_REMOTE_IMAGE_URLS", "false").casefold() != "true":
        return False

    try:
        parsed = urlparse(url)
    except ValueError:
        return False

    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        return False
    if parsed.username or parsed.password:
        return False

    allowed_hosts = {
        host.strip().casefold()
        for host in os.getenv("ALLOWED_IMAGE_HOSTS", "").split(",")
        if host.strip()
    }
    return not allowed_hosts or parsed.hostname.casefold() in allowed_hosts


def validate_file_signature(content: bytes) -> bool:
    """Check common image magic numbers before invoking an image decoder."""

    return any(content.startswith(signature) for signature in SIGNATURES)


def validate_file_type(content: bytes) -> bool:
    """Validate the detected MIME type with libmagic."""

    try:
        return magic.from_buffer(content, mime=True) in ALLOWED_MIME_TYPES
    except Exception:
        return False


async def download_and_validate_image(url: str) -> bytes:
    """Download a public image while limiting redirects, time, and memory."""

    if not validate_url(url):
        raise HTTPException(
            status_code=400,
            detail=(
                "Remote image URLs are disabled or the domain is not allowed. "
                "Submit the image as Base64."
            ),
        )

    timeout = httpx.Timeout(30.0, connect=5.0)
    headers = {"User-Agent": "ColeopteraIdentifier/0.1.0"}
    current_url = url

    async with httpx.AsyncClient(timeout=timeout, follow_redirects=False) as client:
        for redirect_count in range(MAX_REDIRECTS + 1):
            await _assert_public_destination(current_url)
            try:
                async with client.stream("GET", current_url, headers=headers) as response:
                    if response.is_redirect:
                        if redirect_count == MAX_REDIRECTS:
                            raise HTTPException(
                                status_code=400,
                                detail="The URL exceeded the redirect limit.",
                            )
                        location = response.headers.get("location")
                        if not location:
                            raise HTTPException(
                                status_code=400,
                                detail="The URL returned an invalid redirect.",
                            )
                        current_url = urljoin(current_url, location)
                        if not validate_url(current_url):
                            raise HTTPException(
                                status_code=400,
                                detail="The redirect is not allowed.",
                            )
                        continue

                    response.raise_for_status()
                    declared_size = response.headers.get("content-length")
                    if declared_size and int(declared_size) > MAX_FILE_SIZE:
                        raise HTTPException(
                            status_code=413,
                            detail="The image exceeds the 10 MB limit.",
                        )

                    content = bytearray()
                    async for chunk in response.aiter_bytes():
                        content.extend(chunk)
                        if len(content) > MAX_FILE_SIZE:
                            raise HTTPException(
                                status_code=413,
                                detail="The image exceeds the 10 MB limit.",
                            )
                    image_bytes = bytes(content)
                    break
            except httpx.TimeoutException as exc:
                raise HTTPException(
                    status_code=408,
                    detail="Timed out while downloading the image.",
                ) from exc
            except httpx.HTTPStatusError as exc:
                raise HTTPException(
                    status_code=400,
                    detail="The image URL returned an HTTP error.",
                ) from exc
            except httpx.RequestError as exc:
                raise HTTPException(
                    status_code=400,
                    detail="The image could not be downloaded.",
                ) from exc
            except ValueError as exc:
                raise HTTPException(
                    status_code=400,
                    detail="The response contains an invalid size.",
                ) from exc
        else:
            raise HTTPException(status_code=400, detail="The image URL could not be followed.")

    if (
        not image_bytes
        or not validate_file_signature(image_bytes)
        or not validate_file_type(image_bytes)
    ):
        raise HTTPException(
            status_code=400,
            detail="The downloaded content is not a supported image.",
        )
    return image_bytes


async def _assert_public_destination(url: str) -> None:
    """Reject hosts that resolve to loopback, private, or link-local networks."""

    hostname = urlparse(url).hostname
    if not hostname:
        raise HTTPException(status_code=400, detail="The URL does not contain a valid domain.")

    try:
        addresses = await asyncio.to_thread(_resolve_addresses, hostname)
    except socket.gaierror as exc:
        raise HTTPException(
            status_code=400,
            detail="The image domain could not be resolved.",
        ) from exc

    if not addresses or any(not address.is_global for address in addresses):
        raise HTTPException(status_code=400, detail="The URL points to a non-public network.")


def _resolve_addresses(hostname: str) -> set[ipaddress.IPv4Address | ipaddress.IPv6Address]:
    records = socket.getaddrinfo(hostname, None, type=socket.SOCK_STREAM)
    return {ipaddress.ip_address(record[4][0]) for record in records}
