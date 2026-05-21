Let's simplify the codebase a little, with widgets and scenes combining. The whimsy scene should only look for it's next quote/refresh
  on unload, so it's ready to go the next time it's loaded. No need to rotate if it's not called. Doing network on unload means we don't
  get caught waiting for it. (It should log the text it's showing on display, not on unload, so the logs make sense.) 
