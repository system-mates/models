FROM mate/runtime

# Add local models to the /mate/models folder
COPY system/models /mate/models
COPY example/models /mate/models


