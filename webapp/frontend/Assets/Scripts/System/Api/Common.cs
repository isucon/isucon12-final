using System;
using Data;
using UnityEngine.Networking;

namespace Network
{
    [Serializable]
    public class CommonRequest
    {
        public string viewerId;
    }
    
    [Serializable]
    public class UpdatedResource
    {
        public long now;
        public User user;
        public UserDevice userDevice;
        public UserCard[] userCards;
        public UserDeck[] userDecks;
        public UserItem[] userItems;
        public UserLoginBonus[] userLoginBonuses;
        public UserPresent[] userPresents;
    }

    [Serializable]
    public class CommonResponse
    {
        public UpdatedResource updatedResources;
    }

    [Serializable]
    public class ErrorResponse
    {
        public int status_code;
        public string message;
    }

    public abstract class ApiBase<TRequest, TResponse>
        where TRequest : CommonRequest
        where TResponse : CommonResponse
    {
        public virtual string Method => UnityWebRequest.kHttpVerbPOST;
        public abstract string Path { get; }
        public TRequest RequestData { get; protected set; }
    }
}
