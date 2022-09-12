using System;
using System.Threading.Tasks;

namespace Network
{
    [Serializable]
    public class CreateUserRequest : CommonRequest
    {
        public int platformType;
    }

    [Serializable]
    public class CreateUserResponse : CommonResponse
    {
        public long userId;
        public string viewerId;
        public string sessionId;
        public long createdAt;
    }
    
    public class CreateUserApi : ApiBase<CreateUserRequest, CreateUserResponse>
    {
        public override string Path { get; } = "/user";

        public CreateUserApi(string viewerId)
        {
            RequestData = new CreateUserRequest
            {
                viewerId = viewerId,
                platformType = 1
            };
        }
    }

    public partial class ApiClient
    {
        public async Task<CreateUserResponse> CreateUserAsync(string viewerId)
        {
            var api = new CreateUserApi(viewerId);
            var res = await Post(api);
            ViewerId = res.viewerId;
            SessionId = res.sessionId;
            return res;
        }
    }
}
