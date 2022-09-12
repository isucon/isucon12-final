using System;
using System.Threading.Tasks;
using Data;

namespace Network
{
    [Serializable]
    public class LoginRequest : CommonRequest
    {
        public long userId;
    }
    
    [Serializable]
    public class LoginResponse : CommonResponse
    {
        public User user;
        public UserDeck deck;
        public int totalAmountPerSec;
        public long pastTime;
        public long now;
    }
    
    [Serializable]
    public class LoginApi : ApiBase<CommonRequest, LoginResponse>
    {
        public override string Path { get; } = "/login";

        public LoginApi(long userId)
        {
            RequestData = new LoginRequest()
            {
                userId = userId
            };
        }
    }

    public partial class ApiClient
    {
        public async Task<LoginResponse> LoginAsync()
        {
            var api = new LoginApi(GameManager.userData.user.id);
            var res = await Post(api);
            return res;
        }
    }
}
