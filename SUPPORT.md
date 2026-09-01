# Support policy

The latest stable 1.x minor receives bug fixes and security updates. The
immediately previous minor receives security fixes for six months after its
successor is released. Pre-1.0 development builds have no support window.

Generated applications remain independent from Scaffold Agent. Their Go, Java,
Python, Node.js, PostgreSQL, MySQL, and third-party dependency support follows
the versions locked in the generating release. A later Engine may offer an
upgrade Plan, but it does not silently mutate an existing project.

Use public issues for reproducible non-sensitive defects and feature requests.
Use GitHub private vulnerability reporting for security issues. Include the
Engine version, operating system, Blueprint with secrets removed, command or MCP
tool, stable diagnostic code, and minimal reproduction. Do not attach customer
data, credentials, private source, `.scaffold-agent` artifacts, or generated
backups.
