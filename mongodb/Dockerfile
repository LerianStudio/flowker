FROM mongo:7.0

# Create directory and copy keyfile with correct permissions
RUN mkdir -p /etc/mongodb && chown mongodb:mongodb /etc/mongodb
COPY --chown=mongodb:mongodb keyfile /etc/mongodb/keyfile
RUN chmod 400 /etc/mongodb/keyfile
